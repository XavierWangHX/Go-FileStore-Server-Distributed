package handler

import (
	cmn "FileStore/common"
	cfg "FileStore/config"
	"FileStore/db"
	"FileStore/meta"
	"FileStore/mq"
	"FileStore/store/oss"
	"FileStore/util"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func UploadGetHandler(c *gin.Context) {
	c.Redirect(http.StatusFound, "/static/view/upload.html")
}

func UploadPostHandler(c *gin.Context) {
	errCode := 0
	defer func() {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		if errCode < 0 {
			c.JSON(http.StatusOK, gin.H{
				"code": errCode,
				"msg":  "上传失败",
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"code": errCode,
				"msg":  "上传成功",
			})
		}
	}()

	file, head, err := c.Request.FormFile("file")
	if err != nil {
		errCode = -1
		log.Printf("Failed to get data, err: %s", err.Error())
		return
	}
	defer file.Close()
	fileMeta := meta.FileMeta{
		FileName: head.Filename,
		Location: "/home/whx/Desktop/FileStore/data/" + head.Filename,
		UploadAt: time.Now().Format("2006/01/02 15:04:05"),
	}

	newFile, err := os.Create(fileMeta.Location)
	if err != nil {
		errCode = -2
		log.Printf("Failed to create file, err: %s", err.Error())
		return
	}
	defer newFile.Close()

	fileMeta.FileSize, err = io.Copy(newFile, file)
	if err != nil {
		errCode = -3
		log.Printf("Failed to save file, err: %s", err.Error())
		return
	}

	newFile.Seek(0, 0)
	fileMeta.FileSha1 = util.FileSha1(newFile)

	newFile.Seek(0, 0)

	ossPath := "oss/" + fileMeta.FileSha1 + "/" + fileMeta.FileName
	if !cfg.AsyncTransferEnable {
		err = oss.Bucket().PutObject(ossPath, newFile)
		if err != nil {
			log.Println(err.Error())
			errCode = -4
			return
		}
		fileMeta.Location = ossPath
	} else {
		// 写入异步转移任务队列
		data := mq.TransferData{
			FileHash:      fileMeta.FileSha1,
			CurLocation:   fileMeta.Location,
			DestLocation:  ossPath,
			DestStoreType: cmn.StoreOSS,
		}
		pubData, _ := json.Marshal(data)
		pubSuc := mq.Publish(
			cfg.TransExchangeName,
			cfg.TransOSSRoutingKey,
			pubData,
		)

		if !pubSuc {
			log.Println("Publish Failed!")
			return
		}
		log.Println("Publish Success")

	}

	suc := meta.UpdateFileMetaDB(fileMeta)
	if !suc {
		errCode = -5
		log.Println("UpdateFileMetaDB Failed")
	}
	username := c.Request.FormValue("username")
	suc = db.InsertToUserfileTable(username, fileMeta.FileSha1, fileMeta.FileName, fileMeta.FileSize)
	if suc {
		errCode = 0
	} else {
		errCode = -6
	}

}
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// 返回上传html页面

		data, err := ioutil.ReadFile("./static/view/upload.html")
		if err != nil {
			io.WriteString(w, "internel server error")
			return
		}
		io.WriteString(w, string(data))
		// 另一种返回方式:
		// 动态文件使用http.HandleFunc设置，静态文件使用到http.FileServer设置(见main.go)
		// 所以直接redirect到http.FileServer所配置的url
		// http.Redirect(w, r, "/static/view/index.html",  http.StatusFound)
	} else if r.Method == "POST" {
		// 接收文件流及存储到本地目录

		file, head, err := r.FormFile("file")
		if err != nil {
			fmt.Printf("Failed to get data, err: %s", err.Error())
			return
		}
		defer file.Close()
		fileMeta := meta.FileMeta{
			FileName: head.Filename,
			Location: "/home/whx/Desktop/FileStore/data/" + head.Filename,
			UploadAt: time.Now().Format("2006/01/02 15:04:05"),
		}

		newFile, err := os.Create(fileMeta.Location)
		if err != nil {
			fmt.Printf("Failed to create file, err: %s", err.Error())
			return
		}
		defer newFile.Close()

		fileMeta.FileSize, err = io.Copy(newFile, file)
		if err != nil {
			fmt.Printf("Failed to save file, err: %s", err.Error())
			return
		}

		newFile.Seek(0, 0)
		fileMeta.FileSha1 = util.FileSha1(newFile)

		newFile.Seek(0, 0)

		ossPath := "oss/" + fileMeta.FileSha1 + "/" + fileMeta.FileName
		if !cfg.AsyncTransferEnable {
			err = oss.Bucket().PutObject(ossPath, newFile)
			if err != nil {
				fmt.Println(err.Error())
				w.Write([]byte("Upload failed!"))
				return
			}
			fileMeta.Location = ossPath
		} else {
			// 写入异步转移任务队列
			data := mq.TransferData{
				FileHash:      fileMeta.FileSha1,
				CurLocation:   fileMeta.Location,
				DestLocation:  ossPath,
				DestStoreType: cmn.StoreOSS,
			}
			pubData, _ := json.Marshal(data)
			pubSuc := mq.Publish(
				cfg.TransExchangeName,
				cfg.TransOSSRoutingKey,
				pubData,
			)

			if !pubSuc {
				fmt.Println("Publish Failed!")
				return
			}
			fmt.Println("Publish Success")
			log.Println("Publish Success")

		}

		meta.UpdateFileMetaDB(fileMeta)

		r.ParseForm()
		username := r.Form.Get("username")
		suc := db.InsertToUserfileTable(username, fileMeta.FileSha1, fileMeta.FileName, fileMeta.FileSize)
		if suc {
			resp := util.RespMsg{
				Code: 0,
				Msg:  "OK",
				Data: "/static/view/home.html",
			}
			w.Write(resp.JSONBytes())
		} else {
			w.Write([]byte("Upload Failed"))
		}

	}
}

func FileQueryHandler(c *gin.Context) {

	limitCnt, _ := strconv.Atoi(c.Request.FormValue("limit"))
	username := c.Request.FormValue("username")
	//fileMetas, _ := meta.GetLastFileMetasDB(limitCnt)
	userFiles, err := db.QueryUserFileMetas(username, limitCnt)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(userFiles)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

func TryFastUploadHandler(c *gin.Context) {
	// 1. 解析请求参数
	username := c.Request.FormValue("username")
	filehash := c.Request.FormValue("filehash")
	filename := c.Request.FormValue("filename")
	filesize, _ := strconv.Atoi(c.Request.FormValue("filesize"))

	// 2. 从文件表中查询相同hash的文件记录
	fileMeta, err := meta.GetFileMetaDB(filehash)
	if err != nil {
		log.Println(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	// 3. 查不到记录则返回秒传失败
	nil_filemeta := meta.FileMeta{}
	if fileMeta == nil_filemeta {
		resp := util.RespMsg{
			Code: -1,
			Msg:  "秒传失败，请访问普通上传接口",
		}
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
		return
	}

	// 4. 上传过则将文件信息写入用户文件表， 返回成功
	suc := db.InsertToUserfileTable(
		username, filehash, filename, int64(filesize))
	if suc {
		resp := util.RespMsg{
			Code: 0,
			Msg:  "秒传成功",
		}
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
		return
	}
	resp := util.RespMsg{
		Code: -2,
		Msg:  "秒传失败，请稍后重试",
	}
	c.Data(http.StatusOK, "application/json", resp.JSONBytes())
	return
}

func UploadSucHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Upload Finished!")

}

func GetFileMetaHandler(c *gin.Context) {
	filehash := c.Request.FormValue("filehash")
	//fMeta := meta.GetFileMeta(filehash)
	fMeta, err := meta.GetFileMetaDB(filehash)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(fMeta)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

func DownloadHandler(w http.ResponseWriter, r *http.Request) {

	fsha1 := r.Form["filehash"][0]
	fm, err := meta.GetFileMetaDB(fsha1)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	f, err := os.Open(fm.Location)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+fm.FileName+"\"")
	w.Write(data)
}

func FileMetaUpdateHandler(c *gin.Context) {

	opType := c.Request.FormValue("op")
	fileSha1 := c.Request.FormValue("filehash")
	newFileName := c.Request.FormValue("filename")
	if opType != "0" {
		c.Status(http.StatusForbidden)
		return
	}

	curFileMeta, err := meta.GetFileMetaDB(fileSha1)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	curFileMeta.FileName = newFileName
	meta.UpdateFileMetaDB(curFileMeta)
	data, err := json.Marshal(curFileMeta)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusOK)
	c.Data(http.StatusOK, "application/json", data)

}

func FileDeleteHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	fileSha1 := r.Form.Get("filehash")
	fMeta := meta.GetFileMeta(fileSha1)
	os.Remove(fMeta.Location)
	meta.RemoveFileMeta(fileSha1)
	w.WriteHeader(http.StatusOK)

}

func DownloadURLHandler(c *gin.Context) {
	filehash := c.Request.FormValue("filehash")
	// 从文件表查找记录
	row, _ := db.GetFileMeta(filehash)
	signedURL := oss.DownloadURL(row.FileAddr.String)
	c.Data(http.StatusOK, "application/octet-stream", []byte(signedURL))
}
