# monitor
Auto scalling tool for AWS EC2 written with GO

# Installation
## Install GOLANG

set GOPATH

```
$ cat /etc/profile
# /etc/profile: system-wide .profile file for the Bourne shell (sh(1))

# set environment for MTaaS
export GOPATH="/home/ubuntu/monitor"
```
## Install AWS GO SDK

```
$ echo $GOPATH
/home/ubuntu/monitor
$ pwd
/home/ubuntu/monitor/src/github.com/tianhongbo/monitor
ubuntu@ip-172-31-24-190:~/monitor/src/github.com/tianhongbo/monitor$ go get -u github.com/aws/aws-sdk-go/...
```
