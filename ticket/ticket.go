package ticket

import (
	"../timer"
	"./model"
	"crypto/md5"
	"encoding/hex"
	"log"
	"math/rand"
	"strconv"
	"time"
)

type TicketManager struct {
	ticketMap    map[string] *model.TicketData
	ticketInvalidTime  int
}

func (tm *TicketManager) GenerateTicket(data *model.TicketData) string {
	ticket := MD5(data.URL + strconv.FormatInt(time.Now().UTC().UnixNano(), 10) + strconv.FormatFloat(rand.Float64(), 'f', 2, 64))[:16]

	data.StartTimeStamp = time.Now().Unix()
	data.ExpireTimeStamp = data.StartTimeStamp + (int64)(tm.ticketInvalidTime)
	tm.ticketMap[ticket] = data
	timer.SetTimer("ticket_timeout", (uint32)(tm.ticketInvalidTime), tm.timeout, ticket)
	//log.Println("New Ticket ->", tm.ticketMap[ticket])
	return ticket
}

func (tm *TicketManager) timeout(args interface{}) {
	ticket := args.(string)
	if len(ticket) > 0 && tm.hasTicket(ticket) {
		log.Println("Ticket超时删除 ->", ticket)
		tm.DestroyTicket(ticket)
	}
}

func (tm *TicketManager) hasTicket(ticket string) bool {
	return tm.ticketMap[ticket] != nil
}

func (tm *TicketManager) DestroyTicket(ticket string) {
	delete(tm.ticketMap, ticket)
}

func (tm *TicketManager) IsInvalidTicket(ticket string) bool {
	//log.Println(data.StartTimeStamp + (int64)(tm.ticketInvalidTime), time.Now().UTC().Unix())
	destroyTime := tm.GetDestroyTime(ticket)
	return destroyTime != -1 && destroyTime > time.Now().UTC().Unix()
}

func (tm *TicketManager) GetDestroyTime(ticket string) (int64) {
	data := tm.ticketMap[ticket]
	if data == nil {
		return -1
	}

	return data.ExpireTimeStamp
}

func (tm *TicketManager) GetTicket(ticket string) *model.TicketData {
	return tm.ticketMap[ticket]
}

func MD5(text string) string {
	ctx := md5.New()
	ctx.Write([]byte(text))
	return hex.EncodeToString(ctx.Sum(nil))
}

func NewTicketManager(ticketInvalidTimeSecond int) *TicketManager {
	go timer.Run()
	return &TicketManager{
		ticketMap:            make(map[string] *model.TicketData),
		ticketInvalidTime:    ticketInvalidTimeSecond,
	}
}


