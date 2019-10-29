package model

type TicketData struct {
	URL                string  //文件URL
	FileName           string  //自定义文件名
	Headers            []map[string]interface{}  //自定义请求头
	Cookie             string  //自定义Cookies
	StartTimeStamp     int64  //创建任务开始时间戳
	ExpireTimeStamp    int64  //Token过期时间戳
	DownloadCounter    int64  //下载次数计数
}