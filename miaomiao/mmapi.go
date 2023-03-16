package miaomiao

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	logconfig "miaomiao/miaomiao/config/log"
	"miaomiao/miaomiao/config/municipality"
	"miaomiao/miaomiao/util/fileutil"
	"miaomiao/miaomiao/util/parseutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
    // "sync"
	log "github.com/sirupsen/logrus"

	"github.com/liushuochen/gotable"
)
// GET https://miaomiao.scmttec.com/seckill/seckill/log.do?id=5571 HTTP/1.1
// Host: miaomiao.scmttec.com
// Connection: keep-alive
// X-Requested-With: XMLHttpRequest
// tk: wxapptoken:10:78c1ab171facf32d119490d23fff0fca_2e31a5c13b7fdadc54dac859c4177917
// user-agent: Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.102 Safari/537.36 MicroMessenger/7.0.20.1781(0x6700143B) NetType/WIFI MiniProgramEnv/Windows WindowsWechat/WMPF XWEB/6500
// Content-Type: application/json
// Accept: application/json, text/plain, */*
// referer: https://servicewechat.com/wxff8cad2e9bf18719/36/page-frame.html
// xweb_xhr: 1
// Cookie: _xxhm_=%7B%22id%22%3A37979877%2C%22mobile%22%3A%2217764838748%22%2C%22nickName%22%3A%22%E8%B0%A2%E4%BD%99%E5%8D%8E+%28%E4%BF%A1%E6%81%AF%E9%83%A8-%E5%90%8E%E7%AB%AF%E5%BC%80%E5%8F%91%29%22%2C%22headerImg%22%3A%22https%3A%2F%2Fthirdwx.qlogo.cn%2Fmmopen%2Fvi_32%2FolhhB8ndD8iaa2n2ptjhicllGPbwZ5IwsAvpmHV6XHSuetTkf2gGRMmsyxE7ayb8Q2ZhGqPqtrLic0wN1fqd9Fl3g%2F132%22%2C%22regionCode%22%3A%22500243%22%2C%22name%22%3A%22%E8%B0%A2*%E5%8D%8E%22%2C%22birthday%22%3A%221994-12-15+02%3A00%3A00%22%2C%22sex%22%3A1%2C%22hasPassword%22%3Afalse%2C%22birthdayStr%22%3A%221994-12-15%22%7D; _xzkj_=wxapptoken%3A10%3A78c1ab171facf32d119490d23fff0fca_2e31a5c13b7fdadc54dac859c4177917; 1e0a=8a1013960421b98164; 20da=887bcb3d427d4f3acb; a80e=a107df8c37701f4339; tgw_l7_route=310b1314d3b7b84666fb433380f2a0d4
// Sec-Fetch-Site: cross-site
// Sec-Fetch-Mode: cors
// Sec-Fetch-Dest: empty
// Accept-Encoding: gzip, deflate, br
// Accept-Language: zh-CN,zh


const (
	baseUrl     = "https://miaomiao.scmttec.com"
	userAgent   = "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.102 Safari/537.36 MicroMessenger/7.0.20.1781(0x6700143B) NetType/WIFI MiniProgramEnv/Windows WindowsWechat/WMPF XWEB/6500"
	referer     = "https://servicewechat.com/wxff8cad2e9bf18719/36/page-frame.html"
	accept      = "application/json, text/plain, */*"
	host        = "miaomiao.scmttec.com"
	contentType = "application/json"
)

var (
	// Regions 区域数据
	Regions = make([]Region, 0)
)

const (
	// 秒苗小程序内固定的 md5 加密 salt
	salt = "ux$ad70*b"
	// HeaderFilePath 抓包信息文件路径
	HeaderFilePath = "./header.txt"
)

// Miaomiao 秒苗
type Miaomiao struct {
	BaseUrl string
	Headers map[string]interface{}
	Apis    Apis
}

func init() {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("启动程序失败，原因：日志切割配置出错，error: %+v", err)
			ExitAfterSleep(15 * time.Second)
		}
	}()
	logconfig.CustomLogFormat()
	if !fileutil.IsExist(HeaderFilePath) {
		if err := ioutil.WriteFile(HeaderFilePath, []byte{}, 0644); err != nil {
			log.Errorf("启动失败：生成 header.txt 文件失败")
			ExitAfterSleep(15 * time.Second)
		}
		log.Errorf("启动失败：缺少 header.txt 文件，已自动生成 header.txt 文件，请将抓包数据填写进去后重启该软件")
		ExitAfterSleep(15 * time.Second)
	}
}

// Start .
func Start() {
	m := &Miaomiao{
		BaseUrl: baseUrl,
		Headers: map[string]interface{}{
			"user-agent":   userAgent,
			"referer":      referer,
			"Accept":       accept,
			"Host":         host,
    		"xweb_xhr": "1",                  // 固定为 1
			"Content-Type": contentType,
			"Cookie":       make(map[string]string),
		},
		Apis: Apis{
			Log: Api{
			    Url:"/seckill/seckill/log.do",
			    Method:http.MethodGet,
			},
			ChildRegions: Api{
				Url:    "/base/region/childRegions.do",
				Method: http.MethodGet,
			},
			Vaccines: Api{
				Url:    "/seckill/seckill/list.do",
				Method: http.MethodGet,
			},
			Member: Api{
				Url:    "/seckill/linkman/findByUserId.do",
				Method: http.MethodGet,
			},
			St: Api{
				Url:    "/seckill/seckill/checkstock2.do",
				Method: http.MethodGet,
			},
			Subscribe: Api{
				Url:    "/seckill/seckill/subscribe.do",
				Method: http.MethodGet,
			},
		},
	}
	
	// 显示解析出来的 cookie 和 tk
	m.showCookieAndTk()
	// 初始化地区数据
	m.initRegion()

	// 获取并选择接种人信息
	selectedMemberId, selectedIdCard := m.showAndSelectMember()
	// 选择省
	selectedRegionCode, selectedRegionName := m.showAndSelectParentRegion()

	// 判断选择的是否是直辖市
	isMunicipality := false
	for _, mp := range municipality.Municipalities {
		if mp.Code == selectedRegionCode {
			isMunicipality = true
			break
		}
	}
	// 如果是非直辖市，则还需要根据选择的省级编码选择具体的市级编码
	if !isMunicipality {
		selectedRegionCode, selectedRegionName = m.showAndSelectChildRegion(selectedRegionCode, selectedRegionName)
	}


	
	// 获取该地区的疫苗列表
	selectedVaccineId, killTime := m.showAndSelectVaccine(selectedRegionCode, selectedRegionName)

	// 提前将需要类型转换的变量转换成需要的形式，减少请求过程中的类型转换操作
	selectedVaccineIdStr := strconv.FormatInt(selectedVaccineId, 10)
	selectedMemberIdStr := strconv.Itoa(int(selectedMemberId))

	// 计算秒杀开始时间的毫秒级时间戳
	loc, err := time.LoadLocation("Local")
	if err != nil {
		log.Errorf("调用 LoadLocation 失败: %+v", err)
		ExitAfterSleep(15 * time.Second)
	}
	locTime, err := time.ParseInLocation("2006-01-02 15:04:05", killTime, loc)
	if err != nil {
		log.Errorf("将%s转换为时间戳失败：%+v", killTime, err)
		ExitAfterSleep(15 * time.Second)
	}

	
	fmt.Printf("请输入提前开始时间(毫秒)：")
    var starttime int64
	for {
		if _, err = fmt.Scanln(&starttime); err != nil {
			fmt.Printf("输入信息有误，输入提前开始时间 starttime: ")
		}
		isAvailable := false
		if starttime>0 {
			log.Infof("已选择提前开始时：%d", starttime)
			isAvailable = true
		}
		if isAvailable {
			break
		}
	}
	
	fmt.Printf("输入间隔时间(毫秒 300 500 1500)：")
	var sleeptime int64
	for {
		if _, err = fmt.Scanln(&sleeptime); err != nil {
			fmt.Printf("输入信息有误，输入间隔时间 sleeptime: ")
		}
		isAvailable := false
		if sleeptime>0 {
			log.Infof("输入间隔时间：%d", sleeptime)
			isAvailable = true
		}
		if isAvailable {
			break
		}
	}
	
	// 毫秒级时间戳
	killTimestampMilli := locTime.UnixMilli()
	// 提示距离秒杀开始还有多少秒
	log.Infof("距离秒杀开始还有%d秒，请不要关闭本系统，等待秒杀开始即可", (killTimestampMilli-time.Now().UnixMilli())/1000)

    
	for {
		// 提前一点时间发送请求，这里的 288 是我本地多次调整的一个经验数字，299需要根据自己的网络环境自己调整，单位是毫秒
		if killTimestampMilli-time.Now().UnixMilli() <= starttime {
		    success := false
		    
		  //  m.getLog(selectedVaccineIdStr) //1500
		  //  time.Sleep(300 * time.Millisecond)
		    st := m.getSt(selectedVaccineIdStr)
            time.Sleep(time.Duration(sleeptime) * time.Millisecond)
            
		
			// 这个循环是为了解决本地服务器时间和秒苗服务器时间不一致的问题
			// 一直循环获取 st 直到 st 值大于等于秒杀开始时间，否则会提示秒杀还未开始
			// 目前 st 接口没有请求频率限制，不需要做延时但是请求多了后会出现返回 0 的问题
// 			for st < killTimestampMilli {
// 				st = m.getSt(selectedVaccineIdStr)
// 			}
            // time.Sleep(time.Duration(10) * time.Millisecond)
            // time.Sleep(50 * time.Millisecond)
            
			eccHs := m.EccHs(selectedVaccineIdStr, selectedMemberIdStr, strconv.FormatInt(st, 10))
			if m.secKill(selectedVaccineIdStr, selectedMemberIdStr, selectedIdCard, eccHs) {
				// 秒杀成功就退出循环了
				// break
				success = true
			} 
			
			
// 			time.Sleep(time.Duration(100) * time.Millisecond)
		
        	if success {
        	    break
        	}
		}
    	// 目前是睡眠 5ms，可以自己调整
    	time.Sleep(5 * time.Millisecond)
		
	}




// 	st := m.getSt(selectedVaccineIdStr)
// 	eccHs := m.EccHs(selectedVaccineIdStr, selectedMemberIdStr, strconv.FormatInt(st, 10))
// 	log.Warnf("selectedMemberIdStr %s =>:%s", strconv.FormatInt(st, 10), eccHs)
	
// 	m.secKill(selectedVaccineIdStr, selectedMemberIdStr, selectedIdCard, eccHs)

	// 实现一：直接 for 循环 + sleep 来轮询
// 	for {
// 		// 提前一点时间发送请求，这里的 288 是我本地多次调整的一个经验数字，299需要根据自己的网络环境自己调整，单位是毫秒
// 		if killTimestampMilli-time.Now().UnixMilli() <= starttime {
// 		    success := false
//         	wg := sync.WaitGroup{}
//         	wg.Add(5)
//         	for i := 0; i < 5; i++ {
//             	go func() {
//         			st := m.getSt(selectedVaccineIdStr)
//         			// 这个循环是为了解决本地服务器时间和秒苗服务器时间不一致的问题
//         			// 一直循环获取 st 直到 st 值大于等于秒杀开始时间，否则会提示秒杀还未开始
//         			// 目前 st 接口没有请求频率限制，不需要做延时但是请求多了后会出现返回 0 的问题
//         			for st < killTimestampMilli {
//         				st = m.getSt(selectedVaccineIdStr)
//         			}
//         			eccHs := m.EccHs(selectedVaccineIdStr, selectedMemberIdStr, strconv.FormatInt(st, 10))
//         			if m.secKill(selectedVaccineIdStr, selectedMemberIdStr, selectedIdCard, eccHs) {
//         				// 秒杀成功就退出循环了
//         				// break
//         				success = true
//         			} 
//         			time.Sleep(time.Duration(200) * time.Millisecond)
//         			wg.Done()
//             	}()
//         	}
//         	if success {
//         	    break
//         	}
//         	wg.Wait()
// 		}
//     	// 目前是睡眠 5ms，可以自己调整
//     	time.Sleep(5 * time.Millisecond)
		
// 	}

	// 实现二：使用定时器 ticker 来轮询
	//// 10ms 检测一次是否到达秒杀时间
	//ticker := time.NewTicker(5 * time.Millisecond)
	//go func() {
	//	st := m.getSt(selectedVaccineIdStr)
	//	for range ticker.C {
	//		// 这里的提前时间需要根据自己的时间和服务器时间的差距进行微调，目前未秒杀时请求 st 的总耗时平均在 55ms 左右
	//		if killTimestampMilli-time.Now().UnixMilli() <= 200 {
	//			// 这个循环是为了解决本地服务器时间和秒苗服务器时间不一致的问题
	//			// 一直循环获取 st 直到 st 值大于等于秒杀开始时间，否则会提示秒杀还未开始
	//			// 目前 st 接口没有请求频率限制，不需要做延时但是请求多了后会出现返回 0 的问题
	//			for st < killTimestampMilli {
	//				st = m.getSt(selectedVaccineIdStr)
	//			}
	//			eccHs := m.EccHs(selectedVaccineIdStr, selectedMemberIdStr, strconv.FormatInt(st, 10))
	//			if m.secKill(selectedVaccineIdStr, selectedMemberIdStr, selectedIdCard, eccHs) {
	//				// 秒杀成功就退出循环了
	//				break
	//			}
	//		}
	//	}
	//}()

	// 程序启动后等待 30 分钟即自动关闭该程序，可自己调整挂机时间
	time.Sleep(10 * time.Minute)
}

// Request 发送请求
func (m *Miaomiao) Request(api Api, params map[string]interface{}) (resp *Response, err error) {
	url := m.BaseUrl + api.Url
	// 添加参数
	// 如果是 POST 请求 postBody != nil
	// 如果是 GET 请求 postBody == nil，查询参数直接拼接在 url 后面
	var postBody io.Reader = nil
	switch api.Method {
	case http.MethodGet:
		var queryParam string
		var i = 0
		for k, v := range params {
			if i == 0 {
				queryParam += fmt.Sprintf("?%s=%s", k, v)
			} else {
				queryParam += fmt.Sprintf("&%s=%s", k, v)
			}
			i++
		}
		url += queryParam
	case http.MethodPost:
		byteArr, err := json.Marshal(params)
		if err != nil {
			log.Errorf("序列化 post 请求参数失败：%+v", err)
			ExitAfterSleep(15 * time.Second)
		}
		postBody = bytes.NewReader(byteArr)
	default:
		byteArr, err := json.Marshal(params)
		if err != nil {
			log.Errorf("序列化 post 请求参数失败：%+v", err)
			ExitAfterSleep(15 * time.Second)
		}
		postBody = bytes.NewReader(byteArr)
	}

	client := &http.Client{}
	request, err := http.NewRequest(api.Method, url, postBody)
	if err != nil {
		return nil, err
	}
	
	// 添加请求头信息
	for k, v := range m.Headers {
	    //
		if k == "Cookie" {
			cookieStr := ""
			for ck, cv := range v.(map[string]string) {
				cookieStr = fmt.Sprintf("%s%s=%s; ", cookieStr, ck, cv)
				request.Header.Add(k, cookieStr)
			}
			//
			log.Warnf("cookie  %s", cookieStr)
		} else {
		    log.Warnf("Header  %s => %s", k, v.(string))
			request.Header.Add(k, v.(string))
		}
	}

	response, _ := client.Do(request)
	// 保证最后正确关闭连接
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	headers := response.Header
	if len(headers) > 0 {
		for k, v := range headers {
			if k == "Set-Cookie" {
				for _, cookieStr := range v {
					cookie := strings.TrimSpace(strings.Split(cookieStr, ";")[0])
					
					log.Warnf("Set-Cookie: %s", cookieStr)
					
					kv := strings.Split(cookie, "=")
					m.Headers["Cookie"].(map[string]string)[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
				}
			}
		}
	}
	
	

	resp = &Response{}
	if err = json.Unmarshal(body, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// EccHs 根据疫苗 id 和 加密参数 st 生成 ecc-hs
func (m *Miaomiao) EccHs(vaccineId string, memberId string, st string) (eccHs string) {
	hash := md5.New()
	hash.Write([]byte(vaccineId + memberId + st))
	s := hex.EncodeToString(hash.Sum(nil))
	hash.Reset()
	hash.Write([]byte(s + salt))
	eccHs = hex.EncodeToString(hash.Sum(nil))

	return eccHs
}

// InitParentRegion 初始化父级区域数据
func (m *Miaomiao) InitParentRegion() (ok bool, err error) {
	return m.GetChildRegionsByParentCode("")
}

// GetChildRegionsByParentCode 根据 parentCode 查询子级区域，parentCode 为 "" 即查询的是省级区域
func (m *Miaomiao) GetChildRegionsByParentCode(parentCode string) (ok bool, err error) {
	resp, err := m.Request(m.Apis.ChildRegions, map[string]interface{}{
		"parentCode": parentCode,
	})
	if err != nil {
		return false, err
	}
	if childRegions, ok := resp.Data.([]interface{}); ok {
		if len(childRegions) == 0 {
			return false, fmt.Errorf("未获取到 code 为[%s]的子级区域数据", parentCode)
		}
		// 重新初始化
		Regions = make([]Region, 0)
		for _, region := range childRegions {
			if r, ok := region.(map[string]interface{}); ok {
				Regions = append(Regions, Region{
					Code:         r["value"].(string),
					Name:         r["name"].(string),
					ChildRegions: nil,
				})
			}
		}
	}

	m.handleMunicipality()
	return true, nil
}

// ExitAfterSleep 睡眠指定秒数后退出系统
func ExitAfterSleep(sec time.Duration) {
	log.Warnf("%d秒后将自动关闭该软件", sec/time.Second)
	time.Sleep(sec)
	os.Exit(0)
}

// -------------------------------------- NOT EXPORTED METHOD
// 直辖市数据特殊处理
func (m *Miaomiao) handleMunicipality() {
	// 如果是直辖市则其子级区域只取第一个且 code 只要前四位
	for ir := range Regions {
		for im := range municipality.Municipalities {
			// 如果是直辖市则只取第一个子区域的前 4 位为其区域 code
			if municipality.Municipalities[im].Code[:2] == Regions[ir].Code {
				region := Regions[0]
				region.Code = municipality.Municipalities[im].Code
				region.Name = municipality.Municipalities[im].Name
				Regions[ir] = region
				break
			}
		}
	}
}

// 显示解析的 cookie 和 tk
func (m *Miaomiao) showCookieAndTk() {
	// 解析 Cookie 和 tk
	cookie, tk := parseutil.ExtractCookieAndTkFromHeader(fileutil.ReadAllText(HeaderFilePath))
	if len(cookie) == 0 || len(tk) == 0 {
		log.Errorf("解析 header 错误，请检查 header.txt 的抓包数据是否正确")
		ExitAfterSleep(15 * time.Second)
	}
	m.Headers["Cookie"] = cookie
	m.Headers["tk"] = tk
	log.Infof("cookie 为：%s", cookie)
	log.Infof("tk 为：%s", tk)
}

// 加载区域相关数据
func (m *Miaomiao) initRegion() {
	if ok, err := m.InitParentRegion(); !ok {
		log.Errorf("初始化省级区域失败: %+v", err)
		ExitAfterSleep(15 * time.Second)
	}
}

// 显示接种人列表并选择接种人
func (m *Miaomiao) showAndSelectMember() (selectedMemberId int32, selectedIdCard string) {
	resp, err := m.Request(m.Apis.Member, nil)
	if err != nil {
		log.Errorf("获取接种人信息失败：%+v", err)
		ExitAfterSleep(15 * time.Second)
	}
	if resp.NotOk {
		log.Errorf("获取接种人信息失败：%s", resp.Msg)
		ExitAfterSleep(15 * time.Second)
	}
	// 所有的接种人
	members := make([]Member, 0)
	if data, ok := resp.Data.([]interface{}); ok {
		if len(data) == 0 {
			log.Errorf("未获取到成员信息，请确认是否在秒苗小程序绑定了接种人信息")
			ExitAfterSleep(15 * time.Second)
		}
		for _, item := range data {
			if mem, ok := item.(map[string]interface{}); ok {
				members = append(members, Member{
					Id:     int32(mem["id"].(float64)),
					Name:   mem["name"].(string),
					IdCard: mem["idCardNo"].(string),
				})
			}
		}
	}
	t, err := gotable.Create("接种人 id", "接种人姓名", "接种人身份证号")
	if err != nil {
		log.Errorf("创建接种人表格失败：%+v", err)
		ExitAfterSleep(15 * time.Second)
	}
	for _, member := range members {
		if err = t.AddRow([]string{strconv.Itoa(int(member.Id)), member.Name, member.IdCard}); err != nil {
			log.Errorf("创建接种人表格失败：%+v", err)
			ExitAfterSleep(15 * time.Second)
		}
	}
	fmt.Printf("%v", t)
	fmt.Printf("请输入接种人 id:")
	for {
		if _, err = fmt.Scanln(&selectedMemberId); err != nil {
			fmt.Printf("输入信息有误，请重新输入接种人 id: ")
		}
		// 判断选择的接种人 id 是否存在
		isAvailable := false
		for _, member := range members {
			if member.Id == selectedMemberId {
				selectedIdCard = member.IdCard
				log.Infof("已选择接种人：%d-%s-%s", selectedMemberId, member.Name, member.IdCard)
				isAvailable = true
				break
			}
		}
		if isAvailable {
			break
		}
		fmt.Printf("输入信息不合法，请重新输入接种人 id：")
	}
	return selectedMemberId, selectedIdCard
}

// 显示并选择一级区域
func (m *Miaomiao) showAndSelectParentRegion() (selectedRegionCode, selectedRegionName string) {
	t, err := gotable.Create("地区名称", "地区编码")
	if err != nil {
		log.Errorf("创建省/直辖市表格失败：%+v", err)
		ExitAfterSleep(15 * time.Second)
	}
	for _, region := range Regions {
		if err := t.AddRow([]string{region.Name, region.Code}); err != nil {
			log.Errorf("创建省/直辖市表格失败：%+v", err)
			ExitAfterSleep(15 * time.Second)
		}
	}
	fmt.Printf("%v", t)
	fmt.Printf("请输入省/直辖市的地区编码：")
	for {
		_, _ = fmt.Scanln(&selectedRegionCode)
		selectedRegionCode = strings.TrimSpace(selectedRegionCode)
		// 判断 selectedRegionCode 是否存在
		isAvailable := false
		for _, region := range Regions {
			if region.Code == selectedRegionCode {
				selectedRegionName = region.Name
				log.Infof("您选择的区域为：%s", selectedRegionName)
				isAvailable = true
				break
			}
		}
		if isAvailable {
			break
		}
		fmt.Printf("输入信息不合法，请重新选择地区：")
	}
	return selectedRegionCode, selectedRegionName
}

// 显示并选择二级区域
func (m *Miaomiao) showAndSelectChildRegion(selectedRegionCode string, selectedRegionName string) (string, string) {
	if ok, err := m.GetChildRegionsByParentCode(selectedRegionCode); !ok {
		log.Errorf("获取%s的市级区域失败：%+v", selectedRegionName, err)
		ExitAfterSleep(15 * time.Second)
	}
	t, err := gotable.Create("地区名称", "地区编码")
	if err != nil {
		log.Errorf("创建地级市表格失败：%+v", err)
		ExitAfterSleep(15 * time.Second)
	}
	for _, region := range Regions {
		if err = t.AddRow([]string{region.Name, region.Code}); err != nil {
			log.Errorf("创建地级市表格失败：%+v", err)
			ExitAfterSleep(15 * time.Second)
		}
	}
	fmt.Printf("%v", t)
	fmt.Printf("请输入地级市的地区编码：")
	for {
		_, _ = fmt.Scanln(&selectedRegionCode)
		selectedRegionCode = strings.TrimSpace(selectedRegionCode)
		// 判断 selectedRegionCode 是否存在
		isAvailable := false
		for _, region := range Regions {
			if region.Code == selectedRegionCode {
				selectedRegionName = fmt.Sprintf("%s-%s", selectedRegionName, region.Name)
				log.Infof("您最终选择的区域为：%s", selectedRegionName)
				isAvailable = true
				break
			}
		}
		if isAvailable {
			break
		}
		fmt.Printf("输入信息不合法，请重新选择地区：")
	}
	return selectedRegionCode, selectedRegionName
}

// 显示并选择疫苗
func (m *Miaomiao) showAndSelectVaccine(selectedRegionCode string, selectedRegionName string) (selectedVaccineId int64, killTime string) {
	resp, err := m.Request(m.Apis.Vaccines, map[string]interface{}{
		"offset":     "0",
		"limit":      "100",
		"regionCode": selectedRegionCode,
	})
	if err != nil {
		log.Errorf("获取%s地区疫苗列表失败：%+v", selectedRegionName, err)
		ExitAfterSleep(15 * time.Second)
	}
	if resp.NotOk {
		log.Errorf("获取%s地区疫苗列表失败：%s", selectedRegionName, resp.Msg)
		ExitAfterSleep(15 * time.Second)
	}
	vaccines := make([]Vaccine, 0)
	if data, ok := resp.Data.([]interface{}); ok {
		if len(data) == 0 {
			log.Errorf("%s暂无秒杀信息", selectedRegionName)
			ExitAfterSleep(15 * time.Second)
		} else {
			for _, vaccine := range data {
				if v, ok := vaccine.(map[string]interface{}); ok {
					vaccines = append(vaccines, Vaccine{
						Address:     v["address"].(string),
						Id:          int64(v["id"].(float64)),
						ImgUrl:      v["imgUrl"].(string),
						Name:        v["name"].(string),
						StartTime:   v["startTime"].(string),
						Stock:       int64(v["stock"].(float64)),
						VaccineCode: v["vaccineCode"].(string),
						VaccineName: v["vaccineName"].(string),
					})
				}
			}
		}
	}

	t, err := gotable.Create("id", "疫苗名称", "医院名称", "秒杀时间", "是否已经结束")
	if err != nil {
		log.Errorf("创建疫苗表格失败：%+v", err)
		ExitAfterSleep(15 * time.Second)
	}
	for _, vaccine := range vaccines {
		stateDesc := "未开始"
		if vaccine.Stock == 0 {
			stateDesc = "已结束"
		}
		if err = t.AddRow([]string{strconv.FormatInt(vaccine.Id, 10), vaccine.VaccineName, vaccine.Address, vaccine.StartTime, stateDesc}); err != nil {
			log.Errorf("创建疫苗表格失败：%+v", err)
			ExitAfterSleep(15 * time.Second)
		}
	}
	fmt.Printf("%v", t)
	fmt.Printf("请输入需要抢购的疫苗 id：")
	for {
		if _, err = fmt.Scanln(&selectedVaccineId); err != nil {
			fmt.Printf("输入信息有误，请重新输入疫苗 id: ")
		}
		// 判断选择的疫苗 id 是否存在
		isAvailable := false
		for _, vaccine := range vaccines {
			if vaccine.Id == selectedVaccineId {
				if vaccine.Stock == 0 {
					isAvailable = false
					break
				}
				killTime = vaccine.StartTime
				log.Infof("已选择疫苗：%d-%s-%s-%s", selectedVaccineId, vaccine.VaccineName, vaccine.Address, vaccine.StartTime)
				isAvailable = true
				break
			}
		}
		if isAvailable {
			break
		}
		fmt.Printf("疫苗不存在或秒杀已结束，请重新输入疫苗 id：")
	}
	return selectedVaccineId, killTime
}

// 获取 st 参数
func (m *Miaomiao) getLog(selectedVaccineIdStr string) (st bool) {
	log.Infof("发送请求获取 log...%s", selectedVaccineIdStr)
	// 获取加密参数 st
	
	resp, err := m.Request(m.Apis.Log, map[string]interface{}{
		"id": selectedVaccineIdStr,
	})

	if err != nil {
		log.Errorf("获取 st 参数失败：%+v", err)
	}
	if resp.NotOk {
		log.Errorf("获取 st 参数失败：%s", resp.Msg)
	}
	
	fmt.Println(resp);

	log.Errorf("获取 log 参数结束")
	return true
}


// 获取 st 参数
func (m *Miaomiao) getSt(selectedVaccineIdStr string) (st int64) {
	log.Infof("发送请求获取 st...")
	// 获取加密参数 st
	resp, err := m.Request(m.Apis.St, map[string]interface{}{
		"id": selectedVaccineIdStr,
	})
	if err != nil {
		log.Errorf("获取 st 参数失败：%+v", err)
	}
	if resp.NotOk {
		log.Errorf("获取 st 参数失败：%s", resp.Msg)
	}
	fmt.Println(resp);
	
	
	if data, ok := resp.Data.(map[string]interface{}); ok {
		if len(data) == 0 {
			log.Errorf("获取 st 参数失败：接口响应结果为空")
		}
		st = int64(data["st"].(float64))
	}
	log.Infof("st 参数获取成功：%d", st)
	return st
}

// 开始秒杀
func (m *Miaomiao) secKill(selectedVaccineIdStr string, selectedMemberIdStr string, selectedIdCard string, eccHs string) (success bool) {
	log.Infof("发送秒杀请求...")
// 	return false
	// 开始秒杀
	resp, err := m.Request(m.Apis.Subscribe, map[string]interface{}{
        "Content-Length": "61",
        "isFormData": "[object Boolean]",
	    "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8",
		"seckillId":    selectedVaccineIdStr, // 疫苗 id
		"linkmanId":    selectedMemberIdStr,  // 接种人 id
		"idCardNo":     selectedIdCard,       // 接种人身份证号
		"ecc-hs":       eccHs,                // 加密参数，每次秒杀前都必须要重新请求一次，st 接口会返回一个与疫苗 id 有关的 cookie，且每次请求那个 cookie 值都会改变
	})
	fmt.Println(resp)
	
	if err != nil {
		log.Errorf("发送秒杀请求失败，：%+v", err)
		return false
	}
	if resp.NotOk {
		log.Errorf("秒杀失败：%s，", resp.Msg)
		return false
	}
	if resp.Ok {
		log.Infof("秒杀成功，订单 id 为：%+v，请尽快登录小程序选择预约时间", resp.Data)
		return true
	}
	log.Errorf("秒杀失败，未知异常，响应信息：%+v", resp)
	return false
}
