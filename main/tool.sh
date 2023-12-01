#!/bin/bash

set -e
# ulimit -n 65536

#自定义配置
GOFISHING_PROJECT_NAME="test"
GOFISHING_SERVER_LIST="
	router_server
	cache_server -port 9000
	login_server -port 9001
	hall_server -port 9004
	gateway_server -port 8302 -proxy localhost
	tick_server
"

cd $(dirname $0)
scriptPid=$$
scriptDir=$(pwd)
timerCmd="* * * * * $scriptDir/tool.sh watchdog"

if [[ -e "env.sh" && -f "env.sh" ]]
then
	source "env.sh"
fi

# GOFISHING_SERVER_LIST转成数组
OLD_IFS="$IFS"
IFS=$'\n'
cmdList=($GOFISHING_SERVER_LIST)
IFS="$OLD_IFS"

# 解析出执行路径
serverList=""
for (( i=0; i<${#cmdList[@]}; i=(i+1) ))
do
	cmdList[$i]=$(echo "${cmdList[$i]}" | sed -e 's/^[[:space:]]*//')
	if [[ -n $serverList ]]
	then
		serverList="$serverList "
	fi
	name=$(echo ${cmdList[$i]} | awk '{print $1}')
	serverList="$serverList$name"
done
serverList=($serverList)


system=$(uname -s)

stopWatchdog() {
	crontab -l | grep -v "$timerCmd" | crontab
	stopProcessByName "tool.sh"

	# 关闭脚本
	ps -ef | grep "$scriptDir/tool.sh watchdog" | grep -v "grep" | awk '{print $2}' | xargs kill -9 &>/dev/null || true
	echo "current crontab list:"
	crontab -l
}

startWatchdog() {
	local oldCmd
	# 测试环境屏蔽邮件通知
	if [[ $GOFISHING_PROJECT_NAME == "test" ]]
	then
		return
	fi
	oldCmd=$(crontab -l | grep -v "$timerCmd" || true)
	oldCmd="$oldCmd
$timerCmd"
	echo "$oldCmd" | crontab
	echo "current crontab list:"
	crontab -l
}

matchProcessByName() {
	local path=`pwd`
	local procName=$1
	local matchPid
	local pid
	local proc

	if [ $system = 'Linux' ]
	then
		for pid in `ps -e | grep $procName | awk '{print $1}' || true`
		do
			local name=`cat /proc/$pid/cmdline 2>/dev/null | sed 's/\x0/\t/g' | sed 's/.\///g' | awk '{print $1}' `
			local dir=`readlink /proc/$pid/cwd 2>/dev/null`
			if [[ $path = $dir && $name = $procName && $pid != $scriptPid ]]
			then
				matchPid="$matchPid $pid"
			fi
		done
	elif [ $system = 'Darwin' ]
	then
		OLD_IFS="$IFS"
		IFS=$'\n'
		procs=($(ps -Eef | grep $name || true | grep -v ^$scriptPid$))
		IFS="$OLD_IFS"
		for proc in "${procs[@]}"
		do
			cwd=$(echo "$proc" | sed 's/ /\n/g' | grep ^PWD= || true)
			pid=$(echo "$proc" | awk '{print $2}')
			name=$(echo "$proc" | awk '{print $8}')
			name=${name#*/}
			cwd=${cwd#PWD=}
			if [[ $path = $cwd && $pid != $scriptPid ]]
			then
				matchPid="$matchPid $pid"
			fi
		done
	fi
	echo $matchPid
}

#关闭当前目录下进程
stopProcessByName() {
	local pid
	local path=`pwd`
	local procName=$1
	local signal=$2
	# echo "try stop $1"
	if [[ -z $signal ]]
	then
		signal="SIGKILL"
	fi

	for pid in `matchProcessByName $procName`
	do
		if [[ -n $pid ]]
		then
			echo "stop $signal ok $path/$procName"
			if [[ $system = 'Linux' ]]
			then
				pstree $pid -p| awk -F "[()]" '{print $2}'| xargs kill -s $signal &>/dev/null || true
			elif [[ $system = 'Darwin' ]]
			then
				kill -9 $pid &> /dev/null || true
			fi

			if [[ $signal = "SIGINT" ]]
			then
				local i
				for (( i=0; i<300; i=(retry+1) ))
				do
					sleep 1
					if [[ -z $(ps -ef | grep $pid | grep -v "grep" || true) ]]
					then
						break
					fi
				done
			fi
		fi
	done
}

stopAllServers() {
	local i

	echo "开始关闭服务..."
	for (( i = 0; i < ${#serverList[@]}; i=(i+1) ))
	do
		name=${serverList[${#serverList[@]}-1-$i]}

		sig="SIGKILL"
		if [[ $name = "bingo_server" || $name = "hall_server" ]]
		then
			sig="SIGINT"
		fi
		stopProcessByName $name $sig
	done
	echo "服务已全部关闭..."
}

if [[ $1 = "stop" ]]
then
	stopWatchdog
	if [[ -n $2 ]]
	then
		stopProcessByName $2
	else
		stopAllServers
	fi
fi

startProcess() {
	local i
	for (( i = 0; i < ${#serverList[@]}; i=(i+1) ))
	do
		if [[ ${serverList[$i]} = $1 ]]
		then
			echo "nohup ./${cmdList[$i]} 1>/dev/null 2>>error.log &"
			nohup ./${cmdList[$i]} 1>/dev/null 2>>error.log &
			if [[ "cache_server" = $1 || "router_server" = $1 || "rpc_server" = $1 ]]
			then
				sleep 2
			fi
		fi
	done
}

startAllServers() {
	echo "start all server"
  for name in "${serverList[@]}"
  do
		startProcess $name
  done

	startWatchdog
}

if [[ $1 = "start" ]]
then
	startAllServers
fi

generateProtobufFile() {
  pbPaths=$(find ../ -name "*.proto" | sed -e 's/\/[^\/]*.proto$//' | sort | uniq)

  OLD_IFS="$IFS"
  IFS=$'\n'
  pbPaths=($pbPaths)
  IFS="$OLD_IFS"

  for pbPath in ${pbPaths[@]}
  do
    echo "生成$pbPath/*.proto文件"
    rm -f $pbPath/*.pb.go
    protoc --proto_path=../ --go-grpc_out=../ --go-grpc_opt=paths=source_relative --go_out=../ --go_opt=paths=source_relative $pbPath/*.proto
  done
}

downloadRemotePackages() {
  local goPath=$GOPATH

  if [[ -z $goPath ]]; then
      goPath=~/go/
  fi

	version=`cat ../go.mod | grep "github.com/guogeer/quasar v1\." | awk '{print $2}'`
	echo "go get quasar@$version"
	go install github.com/guogeer/quasar/...@$version
	cp $goPath/bin/router router_server
	cp $goPath/bin/gateway gateway_server
}

buildAllServers() {
	generateProtobufFile

	echo "开始编译*_server.go"
	# shellcheck disable=SC2045
	for f in `ls *_server.go`
	do
		echo "go build $f"
		go build "$f"
	done
	echo "编译结束"
}

if [[ $1 = "build" ]]
then
	if [[ $2 = "-i" || $2 = "-a" ]]
	then
		downloadRemotePackages
	fi

	if [[ "$2" != "-i" ]]
	then
		buildAllServers
	fi
fi

install_name=game_$(date +%Y-%m-%d).tar
upload_package_name=${install_name}.gz

# 打包程序&配置
setupAllFiles() {
	name=$install_name
	echo 清理文件${name}.gz
	rm -f ${name}.gz
	
	echo "开始打包目录或归档文件"
	for f in "tables.zip" "scripts" "configs"
	do
		if [[ -d $f || $f =~ ^.*.zip$ && -d ${f%.zip} ]]
		then
			echo "add ${f}"
			if [[ $f =~ ^.*.zip$ ]]
			then
				basefile=${f%.zip}
				rm -f $basefile.zip
				zip -q $basefile.zip $basefile/*
			fi

			tar rf $name $f
			if [[ $f =~ ^.*.zip$ ]]
			then
				rm -f $f
			fi
		fi
	done

	echo "开始打包程序"
	# shellcheck disable=SC2045
	for f in `ls`
	do
		if [[ $f =~ ^.*server(.exe)?$ && -f $f ]]
		then
			echo "add ${f}"
			tar rf "$name" "$f"
		fi
	done

	if [[ $1 == "-a" ]]
	then
		echo "add other files"
		tar rf $name config_bak.xml
	fi

	gzip $name
	echo 打包成功
}

if [[ $1 = "setup" ]]
then
	setupAllFiles $2
fi

if [[ $1 = "debug" ]]
then
	stopProcessByName $2
	startProcess $2
fi

if [[ $1 = "pb" ]]
then
	generateProtobufFile
fi

installPackage() {
	local name=$1
	echo "start install $name"

	rm scripts/*.lua
	tar zxvf $name
	if [[ -e configs_bak && -d configs_bak ]]
	then
		echo "NOTE: copy configs_bak to configs"
		cp configs_bak/* configs
	fi
}

if [[ $1 = "get" ]]
then
	name=${install_name}.gz
	rm -f $name
	wget http://localhost/$name
	# installPackage
fi

if [[ $1 = "install" ]]
then
	name=${install_name}.gz

	if [[ -n "$2" ]]
	then
		name=$2
	fi
	installPackage $name
fi

if [[ $1 = "monitor" ]]
then
	if [[ $2 = "-q" ]]
	then
		stopWatchdog
	else
		startWatchdog
	fi
fi

sendMail() {
	local title="[$GOFISHING_PROJECT_NAME] $1"
	local msg=$2
	local sign="hello"
	local cc="123456@qq.com"
	curl --data-urlencode "cc=$cc" --data-urlencode "subject=$title" --data-urlencode "message=$msg" --data-urlencode "sign=$sign" http://localhost:8001/plate/send_email
}

# 检测错误日志，每10s检测1次
# 2020/01/02 01:02:00 player.go:100 [ERROR]
catchError() {
	local day=`date +"%m-%d"`
	# lcoal day="10-16" # TEST
	local sec=`date -d "-10 second" +"%S"`
	local sec10=`expr $sec / 10`
	local now=`date -d "-10 second" +"%Y/%m/%d %H:%M:${sec10}[0-9]"`
	local logs=`find log/ -name "run.log.$day"`
	local match="^${now} \S+ \[ERROR\]"
	# local match="^2020/10/16 00:02:[0-9]{2} \S+ \[ERROR\]" # TEST
	local f
	echo "match: $match"
	for f in $logs
	do
		local last_logs=`tail -n 9999 $f`
		local error_line=`echo "$last_logs" | grep -E -n "$match"`
		local line_no=${error_line%%:*}
		local line_end=`expr $line_no + 20`
		if [[ -n $line_no ]]
		then
			local stack=`echo "$last_logs" | sed -n "${line_no},${line_end}p"`
			echo "stack: $stack"
			sendMail "server error log $f" "$stack"
		fi
	done
}

if [[ $1 = "watchdog" ]]
then
	cd $(dirname $0)

	step=10
	dir=`pwd`
	for (( i = 0; i < 60; i=(i+step) ))
	do 
		dumps=""
		for name in "${serverList[@]}"
		do
			if [[ -z `matchProcessByName $name` ]]
			then
				dumps="$dumps $name"
				startProcess $name
			fi
		done

		now=$(date +"%Y-%m-%d %H:%M:%S")
		if [[ -n $dumps ]]
		then
			dump="服务($dumps)于${now}发生异常，已自动拉起"
			sendMail "dump" "$dump"
		fi

		catchError
	  sleep $step
	done
fi

if [[ $1 = "catch_error" ]]
then
	catchError
fi

if [[ $1 = "deploy" ]]
then
	pem_file=$2
	remote_host=$3
	project_name=$4
	work_wx_key=$5
	branch_name=$6
	path=${remote_host#*:}
	host=${remote_host%:*}

	git checkout $branch_name
	last_commit=$(git log -1 --pretty="%H")

	git pull origin $branch_name
	buildAllServers
	downloadRemotePackages
	setupAllFiles

	ssh -i $pem_file $host "cd $path && rm -f $upload_package_name"
	echo "上传$upload_package_name"
	scp -i $pem_file $upload_package_name $host:$path
	echo "上传tool.sh"
	scp -i $pem_file tool.sh $host:$path
	ssh -i $pem_file $host "cd $path && chmod +x tool.sh && ./tool.sh stop && ./tool.sh install $upload_package_name && ./tool.sh start"

	content=`git log $last_commit..HEAD --pretty=format:'%s by %an' --abbrev-commit --no-merges | awk -F '\n'  '{print "  "NR". " $1}'`
	content=${content//\"/\\\"}
	echo "content: $content"

	if [[ -n $work_wx_key ]]
	then
		curl "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=$work_wx_key" \
		-H 'Content-Type: application/json' \
		-d '
		{
				"msgtype": "text",
				"text": {
					"content": "'$project_name'更新：\n'"$content"'"
				},
				"mentioned_mobile_list": []
		}'
	fi
fi

if [[ $1 = "restart" ]]
then
	stopAllServers
	buildAllServers
	startAllServers
fi
