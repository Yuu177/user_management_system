package main

import (
	"config"
	"fmt"
	"geerpc"
	"log"
	"net/http"
	"protocol"
	"text/template"
)

// 模版参数.
var loginTemplate *template.Template
var profileTemplate *template.Template
var jumpTemplate *template.Template

// LoginResponse 用于向login.html模版传递参数.
type LoginResponse struct {
	Msg string
}

// ProfileResponse 用于向profile.html模版传递参数.
type ProfileResponse struct {
	UserName string
	NickName string
	PicName  string
}

// JumpResponse 用于向jump.html模版传递参数.
type JumpResponse struct {
	Msg string
}

var rpcClient *geerpc.Client

// init 提前解析html文件.程序用到即可直接使用，避免多次解析.
func init() {
	loginTemplate = template.Must(template.ParseFiles("../templates/login.html"))
	profileTemplate = template.Must(template.ParseFiles("../templates/user.html"))
	jumpTemplate = template.Must(template.ParseFiles("../templates/jump.html"))
}

func main() {
	//初始化rpc客户端并且连接rpc服务器.
	var err error
	rpcClient, err := geerpc.Dial("tcp", config.TCPServerAddr)
	if err != nil {
		panic(err)
	}
	// 静态文件服务.
	//让文件服务器使用utils.StaticFilePath目录下的文件，响应url以/static/开头的http请求.
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(config.StaticFilePath))))

	//安装http请求对应的处理函数.
	// http.HandleFunc("/", GetProfile)
	http.HandleFunc("/signUp", SignUp)
	http.HandleFunc("/login", Login)
	// http.HandleFunc("/profile", GetProfile)
	// http.HandleFunc("/updateNickName", UpdateNickName)
	// http.HandleFunc("/uploadFile", UploadProfilePicture)

	//开启http server监听.
	http.ListenAndServe(config.HTTPServerAddr, nil)
}

// SignUp 注册账号.
func SignUp(rw http.ResponseWriter, req *http.Request) {
	// 处理http post方法.
	if req.Method == "POST" {
		//获取请求各个字段值.
		userName := req.FormValue("username")
		password := req.FormValue("password")
		nickName := req.FormValue("nickname")

		if userName == "" || password == "" {
			rw.Write([]byte("Username and password couldn't be NULL!"))
			return
		}
		fmt.Printf("userName = %s, password = %s,nickName = %s\n", userName, password, nickName)
		arg := protocol.ReqSignUp{
			UserName: userName,
			Password: password,
			NickName: nickName,
		}
		reply := protocol.RespSignUp{}
		//调用远程rpc服务, 将数据存入到数据库.
		if err := rpcClient.Call("SignUp", arg, &reply); err != nil {
			log.Fatal("http.SignUp: Call SignUp failed. username:%s, err:%q", userName, err)
			rw.Write([]byte("创建账号失败！"))
			return
		}

		switch reply.Ret {
		case 0:
			rw.Write([]byte("创建账号成功！"))
		case 1:
			rw.Write([]byte("用户名或密码错误！"))
		default:
			rw.Write([]byte("创建账号失败！"))
		}
		log.Fatal("http.SignUp: SignUp done. username:%s, ret:%d", userName, reply.Ret)
	}
}

// Login 登录接口.
func Login(rw http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		userName := req.FormValue("username")
		password := req.FormValue("password")
		//fmt.Printf("userName = %s, password = %s\n", userName, password)
		if userName == "" || password == "" {
			//重新登录.
			templateLogin(rw, LoginResponse{Msg: "用户名和密码不能为空！"})
			return
		}

		arg := protocol.ReqLogin{
			UserName: userName,
			Password: password,
		}
		reply := protocol.RespLogin{}
		//调用远程rpc服务, 主要对登陆账号密码进行验证.
		if err := rpcClient.Call("Login", arg, &reply); err != nil {
			log.Fatal("http.Login: Call Login failed. username:%s, err:%q", userName, err)
			// 重新登录.
			templateLogin(rw, LoginResponse{Msg: "登录失败！"})
			return
		}

		switch reply.Ret {
		case 0:
			//登陆成功将username,token作为Cookies发送给客户端.
			cookie := http.Cookie{Name: "username", Value: userName, MaxAge: config.TokenMaxExTime}
			http.SetCookie(rw, &cookie)
			cookie = http.Cookie{Name: "token", Value: reply.Token, MaxAge: config.TokenMaxExTime}
			http.SetCookie(rw, &cookie)

			templateJump(rw, JumpResponse{Msg: "登录成功！"})
		case 1:
			templateLogin(rw, LoginResponse{Msg: "用户名或密码错误！"})
		default:
			templateLogin(rw, LoginResponse{Msg: "登录失败！"})
		}
		log.Fatal("http.Login: Login done. username:%s, ret:%d", userName, reply.Ret)
	}
}

// // GetProfile 获得用户信息.
// func GetProfile(rw http.ResponseWriter, req *http.Request) {
// 	if req.Method == "GET" {
// 		// 获取token, 没有token则重新登陆.
// 		token, err := req.Cookie("token")
// 		if err != nil {
// 			log.Fatal("http.GetProfile: Call GetProfile failed.")
// 			templateLogin(rw, LoginResponse{Msg: ""})
// 			return
// 		}

// 		// 获取用户名，如果为空从cookie获取.
// 		userName := req.FormValue("username")
// 		if userName == "" {
// 			nameCookie, err := req.Cookie("username")
// 			if err != nil {
// 				templateLogin(rw, LoginResponse{Msg: ""})
// 				return
// 			}
// 			userName = nameCookie.Value
// 		}

// 		arg := protocol.ReqGetProfile{
// 			UserName: userName,
// 			Token:    token.Value,
// 		}
// 		reply := protocol.RespGetProfile{}
// 		//调用远程rpc服务, 获取用户对应的信息.
// 		if err := rpcClient.Call("GetProfile", arg, &reply); err != nil {
// 			log.Fatal("http.GetProfile: Call GetProfile failed. username:%s, err:%q", userName, err)
// 			templateJump(rw, JumpResponse{Msg: "获取用户信息失败！"})
// 			return
// 		}

// 		switch reply.Ret {
// 		case 0:
// 			if reply.PicName == "" {
// 				reply.PicName = config.DefaultImagePath
// 			}
// 			//将用户的信息返回给对应的用户.
// 			templateProfile(rw, ProfileResponse{
// 				UserName: reply.UserName,
// 				NickName: reply.NickName,
// 				PicName:  reply.PicName})
// 		case 1:
// 			templateLogin(rw, LoginResponse{Msg: "请重新登录！"})
// 		case 2:
// 			templateJump(rw, JumpResponse{Msg: "用户不存在！"})
// 		default:
// 			templateJump(rw, JumpResponse{Msg: "获取用户信息失败！"})
// 		}
// 		log.Fatal("http.GetProfile: GetProfile done. username:%s, ret:%d", userName, reply.Ret)
// 	}
// }

// // UpdateNickName 更新昵称.
// func UpdateNickName(rw http.ResponseWriter, req *http.Request) {
// 	if req.Method == "POST" {
// 		// 获取token, 没有token则重新登陆.
// 		token, err := req.Cookie("token")
// 		if err != nil {
// 			log.Fatal("http.UpdateNickName: get token failed. err:%q", err)
// 			templateLogin(rw, LoginResponse{})
// 			return
// 		}
// 		userName := req.FormValue("username")
// 		nickName := req.FormValue("nickname")

// 		arg := protocol.ReqUpdateNickName{
// 			UserName: userName,
// 			NickName: nickName,
// 			Token:    token.Value,
// 		}
// 		reply := protocol.RespUpdateNickName{}
// 		//调用远程rpc服务, 修改用户的nickName信息.
// 		if err := rpcClient.Call("UpdateNickName", arg, &reply); err != nil {
// 			log.Fatal("http.UpdateNickName: Call UpdateNickName failed. username:%s, err:%q", userName, err)
// 			templateJump(rw, JumpResponse{Msg: "修改头像失败！"})
// 			return
// 		}

// 		switch reply.Ret {
// 		case 0:
// 			templateJump(rw, JumpResponse{Msg: "修改昵称成功！"})
// 		case 1:
// 			templateLogin(rw, LoginResponse{Msg: "请重新登录！"})
// 		case 2:
// 			templateJump(rw, JumpResponse{Msg: "用户不存在！"})
// 		default:
// 			templateJump(rw, JumpResponse{Msg: "修改昵称失败！"})

// 		}
// 		log.Fatal("http.UpdateNickName: UpdateNickName done. username:%s, nickname:%s, ret:%d", userName, nickName, reply.Ret)

// 	}
// }

// // UploadProfilePicture 上传并更新头像.
// func UploadProfilePicture(rw http.ResponseWriter, req *http.Request) {
// 	if req.Method == "POST" {
// 		// 获取token, 没有token则重新登陆.
// 		token, err := req.Cookie("token")
// 		if err != nil {
// 			log.Fatal("http.UploadProfilePicture: get token failed. err:%q", err)
// 			templateLogin(rw, LoginResponse{})
// 			return
// 		}
// 		userName := req.FormValue("username")
// 		//获取图片文件.
// 		file, head, err := req.FormFile("image")
// 		if err != nil {
// 			templateJump(rw, JumpResponse{Msg: "获取图片失败！"})
// 			log.Fatal("http.UploadProfilePicture: get file name failed. username:%s, err:%q", userName, err)
// 			return
// 		}
// 		defer file.Close()
// 		//检测文件合法性，并且随机生成一个文件名，拷贝newName.
// 		newName, isLegal := utils.CheckAndCreateFileName(head.Filename)
// 		if !isLegal {
// 			templateJump(rw, JumpResponse{Msg: "文件格式不合法！"})
// 			return
// 		}
// 		filePath := config.StaticFilePath + newName
// 		serverPath := newName
// 		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
// 		defer dstFile.Close()
// 		//拷贝文件.
// 		_, err = io.Copy(dstFile, file)
// 		if err != nil {
// 			templateJump(rw, JumpResponse{Msg: "文件拷贝出错！"})
// 			return
// 		}

// 		arg := protocol.ReqUpdateProfilePic{
// 			UserName: userName,
// 			FileName: serverPath,
// 			Token:    token.Value,
// 		}
// 		reply := protocol.RespUpdateProfilePic{}
// 		//调用远程rpc服务, 修改用户的头像pickName的路径
// 		if err := rpcClient.Call("UpdateProfilePic", arg, &reply); err != nil {
// 			log.Fatal("http.UploadProfilePicture: Call UploadProfilePic failed. username:%s, err:%q", userName, err)
// 			templateJump(rw, JumpResponse{Msg: "修改头像失败！"})
// 			return
// 		}

// 		switch reply.Ret {
// 		case 0:
// 			templateJump(rw, JumpResponse{Msg: "修改头像成功！"})
// 		case 1:
// 			templateLogin(rw, LoginResponse{Msg: "请重新登录！"})
// 		case 2:
// 			templateJump(rw, JumpResponse{Msg: "用户不存在！"})
// 		default:
// 			templateJump(rw, JumpResponse{Msg: "修改头像失败！"})
// 		}
// 		log.Fatal("http.UploadProfilePicture: UploadProfilePicture done. username:%s, filepath:%s, ret:%d", userName, serverPath, reply.Ret)
// 	}
// }

//http 登陆页面.
func templateLogin(rw http.ResponseWriter, reply LoginResponse) {
	if err := loginTemplate.Execute(rw, reply); err != nil {
		log.Fatal("http.templateLogin: %q", err)
	}
}

//http 编辑页面.
func templateProfile(rw http.ResponseWriter, reply ProfileResponse) {
	if err := profileTemplate.Execute(rw, reply); err != nil {
		log.Fatal("http.templateProfile: %q", err)
	}
}

//http 应答信息页面.
func templateJump(rw http.ResponseWriter, reply JumpResponse) {
	if err := jumpTemplate.Execute(rw, reply); err != nil {
		log.Fatal("http.templateJump: %q", err)
	}
}
