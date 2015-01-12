package app

import (
	"encoding/json"
	"github.com/EPICPaaS/go-uuid/uuid"
	"github.com/EPICPaaS/yixinappsrv/db"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const (
	INSERT_FILELINK        = "insert into file_link (id , sender_id,file_id ,file_name,file_url,size,created,updated)  values(?,?,?,?,?,?,?,?)"
	UPDATE_FILELINK_TIME   = "update file_link set updated =? where sender_id =? and file_id =?"
	EXIST_FILELINK         = "select id from file_link where sender_id =? and file_id =?"
	SELECT_EXPIRE_FILELINK = "select  id, file_id from file_link where  updated  < ?"
	UPDATE_USER_AVATAR     = "update user set avatar = ? where id = ?"
)

type FileLink struct {
	Id       string
	SenderId string
	FileId   string
	FileName string
	FileUrl  string
	Size     int
	Created  time.Time
	Updated  time.Time
}

/*保存文件链接信息*/
func SaveFileLinK(fileLink *FileLink) bool {
	tx, err := db.MySQL.Begin()
	if err != nil {
		logger.Error(err)
		return false
	}
	//更新
	if ExistFileLink(fileLink) {
		_, err = tx.Exec(UPDATE_FILELINK_TIME, time.Now().Local(), fileLink.SenderId, fileLink.FileId)
	} else { //新增
		_, err = tx.Exec(INSERT_FILELINK, uuid.New(), fileLink.SenderId, fileLink.FileId, fileLink.FileName, fileLink.FileUrl, fileLink.Size, time.Now().Local(), time.Now().Local())
	}
	if err != nil {
		logger.Error(err)
		if err := tx.Rollback(); err != nil {
			logger.Error(err)
		}
		return false
	}

	if err := tx.Commit(); err != nil {
		return false
	}

	return true
}

/*判断是否存在文件链接记录*/
func ExistFileLink(fileLink *FileLink) bool {
	rows, err := db.MySQL.Query(EXIST_FILELINK, fileLink.SenderId, fileLink.FileId)
	if err != nil {
		logger.Error(err)
		return false
	}

	defer rows.Close()

	if err = rows.Err(); err != nil {
		logger.Error(err)
		return false
	}
	return rows.Next()
}

/*删除weedfs服务器文件*/
func DeleteFile(fileId string) bool {
	var url = "http://115.29.107.77:5083/delete?fid=" + fileId
	resp, err := http.Get(url)
	if err != nil {
		logger.Errorf("delete file fail  [ERROR]-%s", err.Error())
		return false
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("ioutil.ReadAll() failed (%s)", err.Error())
		return false
	}
	var respBody []map[string]interface{}
	if err := json.Unmarshal(body, &respBody); err != nil {
		logger.Errorf("convert to json failed (%s)", err.Error())
		return false
	}
	e, ok := respBody[0]["error"].(string)
	if ok {
		logger.Errorf("delete file fail [ERROR]- %s", e)
		return false
	}
	return true
}

/*定时扫描过期的文件链接，如果过期侧删除该文件记录和文件服务器中的文件*/
func ScanExpireFileLink() {

	subTime, _ := time.ParseDuration("-168h")
	expire := time.Now().Local().Add(subTime)

	rows, err := db.MySQL.Query(SELECT_EXPIRE_FILELINK, expire)
	if err != nil {
		logger.Error(err)
		return
	}

	defer rows.Close()
	if err := rows.Err(); err != nil {
		logger.Error(err)
		return
	}

	var delIds []string
	for rows.Next() {
		var id, fileId string
		if err := rows.Scan(&id, &fileId); err != nil {
			logger.Error(err)
			continue
		}

		//删除文件
		if DeleteFile(fileId) {
			delIds = append(delIds, id)
		}
	}

	/*删除文件记录*/
	if len(delIds) > 0 {
		tx, err := db.MySQL.Begin()
		if err != nil {
			logger.Error(err)
			return
		}
		delSql := "delete  from file_link where  id   in ('" + strings.Join(delIds, "','") + "')"
		_, err = tx.Exec(delSql)
		if err != nil {
			logger.Error(err)
			if err := tx.Rollback(); err != nil {
				logger.Error(err)
			}
			return
		}
		//提交操作
		if err := tx.Commit(); err != nil {
			logger.Error(err)
			return
		}
	}

}

/*获取用户头像，直接返回头像流*/
func (*device) GetUserAvatar(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}

	r.ParseForm()
	userName := r.Form.Get("userName")
	uid := userName[:strings.Index(userName, "@")]

	width := r.FormValue("width")
	height := r.FormValue("height")
	u := getUserByUid(uid)

	if nil != u && len(u.Avatar) == 0 {
		w.Write([]byte("not avatar"))
		return
	}

	u.Avatar = strings.Replace(u.Avatar, ",", "/", 1)
	addr := "http://" + Conf.WeedfsAddr + "/" + u.Avatar + "?width=" + width + "&height=" + height
	resp, err := http.Get(addr)
	if err != nil {
		logger.Error(err)
		w.Write([]byte("not find image"))
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("ioutil.ReadAll() failed (%s)", err)
		w.Write([]byte("server error"))
		return
	}

	w.Write(body)
}

/*保存用户头像信息*/
func (*device) SetUserAvatar(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}
	baseRes := baseResponse{OK, ""}
	body := ""
	res := map[string]interface{}{"baseResponse": &baseRes}
	defer RetPWriteJSON(w, r, res, &body, time.Now())

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		res["ret"] = ParamErr
		logger.Errorf("ioutil.ReadAll() failed (%s)", err.Error())
		return
	}
	body = string(bodyBytes)

	var args map[string]interface{}

	if err := json.Unmarshal(bodyBytes, &args); err != nil {
		baseRes.ErrMsg = err.Error()
		baseRes.Ret = ParamErr
		return
	}

	baseReq := args["baseRequest"].(map[string]interface{})
	// Token 校验
	token := baseReq["token"].(string)
	user := getUserByToken(token)
	if nil == user {
		baseRes.Ret = AuthErr
		baseRes.ErrMsg = "Auth failure "
		return
	}

	responseUpload := args["responseUpload"].(map[string]interface{})
	fileId := responseUpload["fid"].(string)
	fileSuffix := args["fileExtention"].(string)
	if !saveUserAvatar(user.Uid, fileId+"/."+fileSuffix) {
		baseRes.Ret = InternalErr
		return
	}

}

/*修改用户头像*/
func saveUserAvatar(userid, avatar string) bool {

	row := db.MySQL.QueryRow("select avatar from user where id = ?", userid)
	var oldAvatar string
	if err := row.Scan(&oldAvatar); err != nil {
		logger.Error(err)
	}

	//没改变不需要修改
	if avatar == oldAvatar {
		return true
	}

	/*修改用户头像信息*/
	tx, err := db.MySQL.Begin()
	if err != nil {
		logger.Error(err)
		return false
	}
	_, err = tx.Exec(UPDATE_USER_AVATAR, avatar, userid)
	if err != nil {
		logger.Error(err)
		if err := tx.Rollback(); err != nil {
			logger.Error(err)
		}
		return false
	}
	//提交操作
	if err := tx.Commit(); err != nil {
		logger.Error(err)
		return false
	}

	//删除旧头像文件
	if len(oldAvatar) != 0 {
		fileId := oldAvatar[:strings.Index(oldAvatar, "/")]
		if !DeleteFile(fileId) {
			logger.Errorf("delete file fail , file id is %s", fileId)
		}
	}

	return true
}
