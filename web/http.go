// Copyright © 2014 Terry Mao, LiuDing All rights reserved.
// This file is part of gopush-cluster.

// gopush-cluster is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// gopush-cluster is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with gopush-cluster.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"net"
	"net/http"
	"reflect"
	"runtime"
	"time"

	"github.com/EPICPaaS/yixinappsrv/app"
	"github.com/b3log/wide/util"
)

const (
	httpReadTimeout = 30 //seconds
)

// StartHTTP start listen http.
func StartHTTP() {

	// 应用消息服务
	appAppServeMux := http.NewServeMux()
	appAppServeMux.Handle("/app/static/", http.StripPrefix("/app/static/", http.FileServer(http.Dir("static"))))

	appAppServeMux.HandleFunc("/app/client/device/login", apiCallStat(app.Device.Login))
	appAppServeMux.HandleFunc("/app/client/device/addOrRemoveContact", app.Device.AddOrRemoveContact)
	appAppServeMux.HandleFunc("/app/client/device/getMember", app.Device.GetMemberByUserName)
	appAppServeMux.HandleFunc("/app/client/device/getOrgInfo", app.Device.GetOrgInfo)
	appAppServeMux.HandleFunc("/app/client/device/getOrgUserList", app.Device.GetOrgUserList)
	appAppServeMux.HandleFunc("/app/client/device/checkUpdate", app.Device.CheckUpdate)
	appAppServeMux.HandleFunc("/app/client/device/search", app.Device.SearchUser)
	appAppServeMux.HandleFunc("/app/client/device/addApnsToken", app.Device.AddApnsToken)
	appAppServeMux.HandleFunc("/app/client/device/delApnsToken", app.Device.DelApnsToken)
	appAppServeMux.HandleFunc("/app/client/device/setSessionState", app.SessionStat)
	appAppServeMux.HandleFunc("/app/client/device/getApplicationList", app.Device.GetApplicationList)
	appAppServeMux.HandleFunc("/app/client/device/getAppOperationList", app.Device.GetAppOperationList)
	appAppServeMux.HandleFunc("/app/client/device/getUserAvatar", app.Device.GetUserAvatar)
	appAppServeMux.HandleFunc("/app/client/device/setUserAvatar", app.Device.SetUserAvatar)

	appAppServeMux.HandleFunc("/app/client/app/syncOrg", app.App.SyncOrg)
	appAppServeMux.HandleFunc("/app/client/app/syncUser", app.App.SyncUser)
	appAppServeMux.HandleFunc("/app/client/app/syncTenant", app.App.SyncTenant)
	appAppServeMux.HandleFunc("/app/client/app/syncQuota", app.App.SyncQuota)
	appAppServeMux.HandleFunc("/app/client/app/setSessionState", app.SessionStat)
	appAppServeMux.HandleFunc("/app/client/app/getSessions", app.App.GetSession)

	appAppServeMux.HandleFunc("/app/client/app/user/auth", app.App.UserAuth)
	appAppServeMux.HandleFunc("/app/client/app/getOrgUserList", app.App.GetOrgUserList)
	appAppServeMux.HandleFunc("/app/client/app/getOrgList", app.App.GetOrgList)
	appAppServeMux.HandleFunc("/app/client/app/addOrgUser", app.App.AddOrgUser)
	appAppServeMux.HandleFunc("/app/client/app/removeOrgUser", app.App.RemoveOrgUser)

	appAppServeMux.HandleFunc("/app/user/erweima", app.UserErWeiMa)

	for _, bind := range app.Conf.AppBind {
		logger.Infof("start app http listen addr:\"%s\"", bind)
		go httpListen(appAppServeMux, bind)
	}
}

func httpListen(mux *http.ServeMux, bind string) {
	server := &http.Server{Handler: mux, ReadTimeout: httpReadTimeout * time.Second}
	l, err := net.Listen("tcp", bind)
	if err != nil {
		logger.Errorf("net.Listen(\"tcp\", \"%s\") error(%v)", bind, err)
		panic(err)
	}
	if err := server.Serve(l); err != nil {
		logger.Errorf("server.Serve() error(%v)", err)
		panic(err)
	}
}

// apiCallStat 包装了一些请求的公共处理以及 API 调用统计.
//
//  1. panic recover
//  2. request stopwatch
//  3. API calls statistics
func apiCallStat(f func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	handler := stat(f)
	handler = stopwatch(handler)
	handler = panicRecover(handler)

	return handler
}

// common wraps the HTTP Handler for some common processes.
//
//  1. panic recover
//  2. request stopwatch
func common(f func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	handler := panicRecover(f)
	handler = stopwatch(handler)

	return handler
}

// stat 包装了请求统计处理.
//
//  1. 调用统计
func stat(handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// handlerName := getFunctionName(handler)

		if !app.ApiCallStatistics(w, r) {
			return
		}

		handler(w, r)
	}
}

// stopwatch wraps the request stopwatch process.
func stopwatch(handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		defer func() {
			logger.Tracef("[%s] [%s]", r.RequestURI, time.Since(start))
		}()

		handler(w, r)
	}
}

// panicRecover wraps the panic recover process.
func panicRecover(handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer util.Recover()

		handler(w, r)
	}
}

func getFunctionName(function interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(function).Pointer()).Name()
}