package app

import (
	//"database/sql"
	"bytes"
	"encoding/json"
	//"fmt"
	"github.com/EPICPaaS/go-uuid/uuid"
	"github.com/EPICPaaS/yixinappsrv/db"
	"github.com/l2x/golang-chinese-to-pinyin"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const (
	// 根据 id 查询应用记录.
	SelectApplicationById = "SELECT  `id`,`name`,`token`,`type`,`status`,`sort`,`level`,`avatar`,`created`,`updated`,`name_Py`,`name_quanpin`,`description` ,`follow`  FROM `application` WHERE `id` = ?"
	//插入应用信息
	InsertApplication = "INSERT INTO `application`(`id`,`name`,`token`,`type`,`status`,`sort`,`level`,`avatar`,`created`,`updated`,`name_Py`,`name_quanpin`,`description` ,`follow`    ) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)"
	//更新用户信息
	UpdateApplication = "UPDATE  `application`  set  `name` = ?  , `token`=? , `type` = ?  ,`status`=? ,`sort`=? , `level`=?   ,  `avatar` =?  ,  `name_Py`=? ,`name_quanpin` =? , `description` =? , `follow` =?  where id = ? "
	// 查询应用记录.
	//SelectAllApplication = "select t.id, t.name, t.name, t.status, t.sort,t.avatar, t.tenant_id,t.name_py,t.name_quanpin ,t.description, IF( isnull(a.follow) ,t.follow,a.follow)  as follow from (SELECT * from application  where tenant_id = ? ) t left join  app_user a on t.id = a.appId and a.uid = ? "
	SelectAllApplication = "select t.id, t.name, t.name as nickName, t.status, t.sort,t.avatar,t.name_py,t.name_quanpin ,t.description, IF( isnull(a.follow) ,t.follow,a.follow)  as follow from application  t  join  app_user a on t.id = a.appId and a.uid = ? order  by  t.sort "
	// 根据 token 获取应用记录.
	SelectApplicationByToken = "SELECT * FROM `application` WHERE `token` = ?"
	//根据应用ID查询应用操作项列表
	SelectAppOpertionByAppId = "SELECT  `id`, `app_id`, `content`,`action`, `operation_type`,`sort`   FROM `operation` WHERE `app_id` = ?  and  parent_id  is  null  order by  sort "
	//根据操作项父ID查询应用操作项列表
	SelectAppOpertionByParentId = "SELECT `id`, `app_id`, `content`,`action`, `operation_type`,`sort`  FROM `operation` WHERE `parent_id` = ?  order by  sort "
	//插入用户关注的应用
	InsertAppUser = "INSERT INTO `app_user`(`id`,`appid`,`uid`,`follow`) VALUES(?,?,?,?)"
	//删除用户关注应用信息
	UpdateAppUser = "UPDATE  app_user  set follow = ?  where id = ? "
	//查询用户是否关注该企业号
	SelectAppUser = "SELECT `id`,`appid`,`uid`,`follow` FROM `app_user` where appid = ?  and uid = ? "
)

// 应用结构.
type application struct {
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	Token       string    `json:"token"`
	Type        string    `json:"type"`
	Status      int       `json:"status"`
	Sort        int       `json:"sort"`
	Level       int       `json:"level"`
	Avatar      string    `json:"avatar"`
	TenantId    string    `json:"tenantId"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
	PYInitial   string    `json:"pYInitial"`
	PYQuanPin   string    `json:"pYQuanPin"`
	Description string    `json:"description"`
	Follow      string    `json:"follow"`
}

// 应用操作项
type operation struct {
	Id            string `json:"id"`
	AppId         string `json:"appId"`
	Content       string `json:"content"`
	Action        string `json:"action"`
	OperationType string `json:"operationType"`
	Sort          int    `json:"sort"`

	OpertionList []*operation `json:"operationList"`
}

// 用户关注的应用
type userapp struct {
	Id     string `json:"id"`
	AppId  string `json:"appId"`
	UId    string `json:"uid"`
	Follow string `json:"follow"`
}

// 根据 id 查询应用记录.
func getApplication(appId string) (*application, error) {
	row := db.MySQL.QueryRow(SelectApplicationById, appId)

	application := application{}

	if err := row.Scan(&application.Id, &application.Name, &application.Token, &application.Type, &application.Status,
		&application.Sort, &application.Level, &application.Avatar, &application.Created, &application.Updated, &application.PYInitial, &application.PYQuanPin, &application.Description, &application.Follow); err != nil {
		logger.Error(err)
		return nil, err
	}

	if len(application.Avatar) > 0 {
		application.Avatar = strings.Replace(application.Avatar, ",", "/", 1)
		application.Avatar = "http://" + Conf.WeedfsAddr + "/" + application.Avatar
	}

	return &application, nil
}

func getAllApplication(customer_id, tenantId, uid string) ([]*member, error) {
	//改成从数据17yop获取 http://www.17yop.com/yop/phone/login/getRuserApps
	//参数为   tenantId   ruserId
	ret := []*member{}

	EI := GetExtInterface(customer_id, "getRuserApps")
	res, err := http.Get(EI.HttpUrl + "?tenantId=" + tenantId + "&ruuid=" + uid)
	if err != nil {
		return ret, nil
	}
	resBodyByte, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return ret, nil
	}
	var respBody map[string]interface{}
	if err := json.Unmarshal(resBodyByte, &respBody); err != nil {
		return ret, nil
	}
	success, ok := respBody["succeed"].(bool)
	////logger.Infof("结果：%v", respBody)

	if ok && success {
		//更新Application信息
		userMapList := respBody["data"].([]interface{})
		for _, o := range userMapList {
			appMap := o.(map[string]interface{})
			app := application{}

			app.Id = appMap["id"].(string)
			app.Name = appMap["name"].(string)
			app.Token = appMap["appcode"].(string)
			app.Type = "app"
			app.Status = 0
			app.Sort = 0
			app.Level = 0
			app.Avatar = appMap["icon"].(string)
			app.Description = appMap["content"].(string)
			app.Created = time.Now()
			app.Updated = time.Now()

			py := Pinyin.New()
			py.Split = ""
			py.Upper = false
			p, _ := py.Convert(app.Name)

			app.PYInitial = p
			app.PYQuanPin = p
			app.Follow = "1"

			insertApp(&app)

			userApp := new(userapp)
			userApp.Id = uuid.New()
			userApp.AppId = app.Id
			userApp.UId = uid
			userApp.Follow = "1"
			insertUserApp(userApp)
		}

	}

	rows, _ := db.MySQL.Query(SelectAllApplication, uid)
	if rows != nil {
		defer rows.Close()
	}

	for rows.Next() {
		rec := member{}

		if err := rows.Scan(&rec.Uid, &rec.Name, &rec.NickName, &rec.Status, &rec.Sort, &rec.Avatar, &rec.PYInitial, &rec.PYQuanPin, &rec.Description, &rec.Follow); err != nil {
			logger.Error(err)

			return nil, err
		}
		if len(rec.Avatar) > 0 {
			//rec.Avatar = strings.Replace(rec.Avatar, ",", "/", 1)
			rec.Avatar = "http://" + Conf.WeedfsAddr + "/" + rec.Avatar
		}
		rec.UserName = rec.Uid + APP_SUFFIX
		ret = append(ret, &rec)

	}

	return ret, nil
}

//根据appId获取应用的列表项
func getAppOpertionListByAppId(appId string) ([]*operation, error) {
	rows, _ := db.MySQL.Query(SelectAppOpertionByAppId, appId)
	if rows != nil {
		defer rows.Close()
	}
	ret := []*operation{}
	for rows.Next() {
		rec := operation{}

		if err := rows.Scan(&rec.Id, &rec.AppId, &rec.Content, &rec.Action, &rec.OperationType, &rec.Sort); err != nil {
			logger.Error(err)
			return nil, err
		}
		crows, _ := db.MySQL.Query(SelectAppOpertionByParentId, &rec.Id)
		if crows != nil {
			defer crows.Close()
		}
		opertionList := []*operation{}
		for crows.Next() {
			crec := operation{}

			if err := crows.Scan(&crec.Id, &crec.AppId, &crec.Content, &crec.Action, &crec.OperationType, &crec.Sort); err != nil {
				logger.Error(err)
				return nil, err
			}
			opertionList = append(opertionList, &crec)
		}
		rec.OpertionList = opertionList
		ret = append(ret, &rec)
	}

	return ret, nil

}

// 根据 token 查询应用记录.
func getApplicationByToken(token string) (*application, error) {
	row := db.MySQL.QueryRow(SelectApplicationByToken, token)

	application := application{}

	if err := row.Scan(&application.Id, &application.Name, &application.Token, &application.Type, &application.Status,
		&application.Sort, &application.Level, &application.Avatar, &application.TenantId, &application.Created, &application.Updated, &application.PYInitial, &application.PYQuanPin, &application.Description, &application.Follow); err != nil {
		logger.Error(err)

		return nil, err
	}

	return &application, nil
}

/*
*   根据Application获取Member
 */

func (*device) GetApplicationList(w http.ResponseWriter, r *http.Request) {
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
		baseRes.Ret = ParamErr
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
	customer_id := baseReq["customer_id"].(string)
	user := getUserByToken(token)
	if nil == user {
		baseRes.Ret = AuthErr
		baseRes.ErrMsg = "会话超时请重新登录"
		return
	}
	members, err := getAllApplication(customer_id, user.TenantId, user.Uid)
	if err != nil {
		baseRes.ErrMsg = err.Error()
		baseRes.Ret = InternalErr
		return
	}

	res["memberList"] = members
	res["memberCount"] = len(members)
}

/*
* 根据应用信息获取应用的操作项
**/
func (*device) GetAppOperationList(w http.ResponseWriter, r *http.Request) {
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
		baseRes.Ret = ParamErr
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
		baseRes.ErrMsg = "会话超时请重新登录"
		return
	}

	username := args["username"].(string)
	if !strings.HasSuffix(username, APP_SUFFIX) {
		baseRes.Ret = AuthErr
		baseRes.ErrMsg = "username没有以 " + APP_SUFFIX + " 结尾"
		return
	}

	appId := strings.Split(username, APP_SUFFIX)[0]
	opertions, err := getAppOpertionListByAppId(appId)
	if err != nil {
		baseRes.ErrMsg = err.Error()
		baseRes.Ret = InternalErr
		return
	}

	res["operationList"] = opertions
	res["operationCount"] = len(opertions)
}

/*用户关注应用*/
func (*device) UserFollowApp(w http.ResponseWriter, r *http.Request) {
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
	token := baseReq["token"].(string)
	//应用校验
	user := getUserByToken(token)
	if nil == user {
		baseRes.Ret = AuthErr

		return
	}

	appName := args["appname"].(string) //1@app
	if !strings.HasSuffix(appName, APP_SUFFIX) {
		baseRes.Ret = AuthErr
		baseRes.ErrMsg = "appname is error format"
		return
	}

	appId := appName[:strings.Index(appName, "@")]
	userApp := &userapp{
		AppId:  appId,
		UId:    user.Uid,
		Follow: "1",
	}

	if insertUserApp(userApp) {
		//发送一条应用消息告知用户关注了该应用
		application, _ := getApplication(appId)
		data := []byte(`{
				"baseRequest":{"token":"` + application.Token + `"},
				"msgType":103 ,
				"content":"感谢你关注了` + application.Name + `" ,
				"toUserNames":["` + user.Uid + USER_SUFFIX + `"],
				"objectContent":{"appId":"` + appId + `" , "content":"非常感谢你关注了` + application.Name + `"},
				"expire":3600
			}`)
		body := bytes.NewReader(data)
		appPush := "http://" + Conf.AppPush[0] + "/app/client/app/user/push"
		http.Post(appPush, "text/plain;charset=UTF-8", body) //不成功也不管了
		////logger.Infof("%s,%s", Conf.AppPush[0], string(data[:]))

		baseRes.Ret = OK
		baseRes.ErrMsg = "Save app user success"
		return
	} else {
		baseRes.Ret = InternalErr
		baseRes.ErrMsg = "Save app user faild"
		return
	}
}

/*用户取消关注应用*/
func (*device) UserUnFollowApp(w http.ResponseWriter, r *http.Request) {

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
	token := baseReq["token"].(string)
	//应用校验
	user := getUserByToken(token)
	if nil == user {
		baseRes.Ret = AuthErr

		return
	}

	appName := args["appname"].(string) //1@app
	if !strings.HasSuffix(appName, APP_SUFFIX) {
		baseRes.Ret = AuthErr
		baseRes.ErrMsg = "appname is error format"
		return
	}

	appId := appName[:strings.Index(appName, "@")]

	userApp := &userapp{
		AppId:  appId,
		UId:    user.Uid,
		Follow: "0",
	}

	if insertUserApp(userApp) {

		application, _ := getApplication(appId)
		//发送一条应用消息告知用户取消关注了该应用
		data := []byte(`{
						"baseRequest":{"token":"` + application.Token + `"},
						"msgType":104 ,
						"content":"非常感谢你对` + application.Name + `的关注，欢迎下次继续使用" ,
						"toUserNames":["` + user.Uid + USER_SUFFIX + `"],
						"objectContent":{"appId":"` + appId + `" , "content":"非常感谢你对` + application.Name + `的关注，欢迎下次继续使用"}
						
					}`)
		body := bytes.NewReader(data)

		appPush := "http://" + Conf.AppPush[0] + "/app/client/app/user/push"
		http.Post(appPush, "text/plain;charset=UTF-8", body) //不成功也不管了

		baseRes.Ret = OK
		baseRes.ErrMsg = "Delete app user success"
		return
	} else {
		baseRes.Ret = InternalErr
		baseRes.ErrMsg = "Delete app user faild"
		return
	}
}

func getUserApp(appId, uid string) (*userapp, error) {
	row := db.MySQL.QueryRow(SelectAppUser, appId, uid)
	userapp := userapp{}

	if err := row.Scan(&userapp.Id, &userapp.AppId, &userapp.UId, &userapp.Follow); err != nil {
		logger.Error(err)

		return nil, err
	}

	return &userapp, nil
}

func insertUserApp(userApp *userapp) bool {
	userapp, _ := getUserApp(userApp.AppId, userApp.UId)
	if nil == userapp {
		tx, err := db.MySQL.Begin()
		if err != nil {
			logger.Error(err)
			return false
		}
		_, err = tx.Exec(InsertAppUser, uuid.New(), userApp.AppId, userApp.UId, userApp.Follow)
		if err != nil {
			logger.Error(err)
			if err := tx.Rollback(); err != nil {
				logger.Error(err)
			}
			return false
		}

		if err := tx.Commit(); err != nil {
			logger.Error(err)
			return false
		}
	} else {
		tx, err := db.MySQL.Begin()
		if err != nil {
			logger.Error(err)
			return false
		}
		_, err = tx.Exec(UpdateAppUser, userApp.Follow, userapp.Id)
		if err != nil {
			logger.Error(err)
			if err := tx.Rollback(); err != nil {
				logger.Error(err)
			}
			return false
		}

		if err := tx.Commit(); err != nil {
			logger.Error(err)
			return false
		}
	}
	return true
}

func insertApp(app *application) bool {
	tapp, _ := getApplication(app.Id)

	if nil == tapp {
		tx, err := db.MySQL.Begin()
		if err != nil {
			logger.Error(err)
			return false
		}
		_, err = tx.Exec(InsertApplication, app.Id, app.Name, app.Token, app.Type, app.Status, app.Sort, app.Level, app.Avatar, app.Created, app.Updated, app.PYInitial, app.PYQuanPin, app.Description, app.Follow)
		if err != nil {
			logger.Error(err)
			if err := tx.Rollback(); err != nil {
				logger.Error(err)
			}
			return false
		}

		if err := tx.Commit(); err != nil {
			logger.Error(err)
			return false
		}
	} else {
		tx, err := db.MySQL.Begin()
		if err != nil {
			logger.Error(err)
			return false
		}
		//logger.Infof("app.Name[%v], app.Token[%v], app.Type[%v], app.Status[%v], app.Sort[%v], app.Level[%v], app.Avatar[%v], app.PYInitial[%v],app.PYQuanPin[%v], app.Description[%v],app.Follow[%v], app.Id[%v]", app.Name, app.Token, app.Type, app.Status, app.Sort, app.Level, app.Avatar, app.PYInitial, app.PYQuanPin, app.Description, app.Follow, app.Id)

		_, err = tx.Exec(UpdateApplication, app.Name, app.Token, app.Type, app.Status, app.Sort, app.Level, app.Avatar, app.PYInitial, app.PYQuanPin, app.Description, app.Follow, app.Id)
		if err != nil {
			logger.Error(err)
			if err := tx.Rollback(); err != nil {
				logger.Error(err)
			}
			return false
		}

		if err := tx.Commit(); err != nil {
			logger.Error(err)
			return false
		}
	}
	return true
}
