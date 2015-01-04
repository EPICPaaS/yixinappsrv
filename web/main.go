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
	"flag"
	"github.com/EPICPaaS/yixinappsrv/app"
	"github.com/EPICPaaS/yixinappsrv/db"
	"github.com/EPICPaaS/yixinappsrv/perf"
	"github.com/EPICPaaS/yixinappsrv/process"
	"github.com/b3log/wide/log"
	"os"
	"runtime"
	"time"
)

var logger = log.NewLogger(os.Stdout)

func main() {
	var err error
	// Parse cmd-line arguments
	logLevel := flag.String("log_level", "info", "logger level")
	flag.Parse()
	log.SetLevel(*logLevel)

	logger.Infof("yixinappsrv-web ver: \"%s\" start", "1.0.0")

	if err = app.InitConfig(); err != nil {
		logger.Errorf("InitConfig() error(%v)", err)
		return
	}
	//init db config
	if err = db.InitConfig(); err != nil {
		logger.Error("db-InitConfig() error(%v)", err)
		return
	}
	// Set max routine
	runtime.GOMAXPROCS(app.Conf.MaxProc)

	// start pprof http
	perf.Init(app.Conf.PprofBind)

	db.InitDB()
	defer db.CloseDB()

	app.InitRedisStorage()

	// start http listen.
	StartHTTP()
	// init process
	// sleep one second, let the listen start
	time.Sleep(time.Second)
	if err = process.Init(app.Conf.User, app.Conf.Dir, app.Conf.PidFile); err != nil {
		logger.Errorf("process.Init() error(%v)", err)
		return
	}

	defer db.MySQL.Close()

	//初始化配额配置
	app.InitQuotaAll()
	go app.LoadQuotaAll()

	// init signals, block wait signals
	signalCH := InitSignal()
	HandleSignal(signalCH)

	logger.Info("web stop")
}
