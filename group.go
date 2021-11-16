package whatsapp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dgraph-io/ristretto"
)

type GroupMetadata struct {
	ID string `json:"id"`
	Owner string `json:"owner"`
	Subject string `json:"subject"`
	Creation uint64 `json:"creation"`
	Participants []GroupParticipant `json:"participants"`
	SubjectTime int64 `json:"subjectTime"`
	SubjectOwner string `json:"subjectOwner"`
}

type GroupParticipant struct {
	ID string `json:"id"`
	IsAdmin bool `json:"isAdmin"`
	IsSuperAdmin bool `json:"isSuperAdmin"`
}

func (wac *Conn) GetGroupMetaData(jid string, cache *ristretto.Cache) (GroupMetadata, error) {
	meta := GroupMetadata{}
	data := []interface{}{"query", "GroupMetadata", jid}
	cacheKey := fmt.Sprintf("query,GroupMetadata,%s", jid)

	if cache != nil {
		if res, hit := cache.Get(cacheKey); hit {
			meta = res.(GroupMetadata)
			return meta, nil
		}
	}
	
	if ch, err := wac.writeJson(data); err != nil {
		return meta, err
	} else if err := json.Unmarshal([]byte(<-ch), &meta); err != nil {
		return meta, err
	} else if cache != nil {
		cache.SetWithTTL(cacheKey, meta, 256, 12 * time.Hour)
		cache.Wait()
	}

	return meta, nil
}

func (wac *Conn) CreateGroup(subject string, participants []string) (<-chan string, error) {
	return wac.setGroup("create", "", subject, participants)
}

func (wac *Conn) UpdateGroupSubject(subject string, jid string) (<-chan string, error) {
	return wac.setGroup("subject", jid, subject, nil)
}

func (wac *Conn) SetAdmin(jid string, participants []string) (<-chan string, error) {
	return wac.setGroup("promote", jid, "", participants)
}

func (wac *Conn) RemoveAdmin(jid string, participants []string) (<-chan string, error) {
	return wac.setGroup("demote", jid, "", participants)
}

func (wac *Conn) AddMember(jid string, participants []string) (<-chan string, error) {
	return wac.setGroup("add", jid, "", participants)
}

func (wac *Conn) RemoveMember(jid string, participants []string) (<-chan string, error) {
	return wac.setGroup("remove", jid, "", participants)
}

func (wac *Conn) LeaveGroup(jid string) (<-chan string, error) {
	return wac.setGroup("leave", jid, "", nil)
}

func (wac *Conn) GroupInviteLink(jid string) (string, error) {
	request := []interface{}{"query", "inviteCode", jid}
	ch, err := wac.writeJson(request)
	if err != nil {
		return "", err
	}

	var response map[string]interface{}

	select {
	case r := <-ch:
		if err := json.Unmarshal([]byte(r), &response); err != nil {
			return "", fmt.Errorf("error decoding response message: %v\n", err)
		}
	case <-time.After(wac.msgTimeout):
		return "", fmt.Errorf("request timed out")
	}

	if int(response["status"].(float64)) != 200 {
		return "", fmt.Errorf("request responded with %d", response["status"])
	}

	return response["code"].(string), nil
}

func (wac *Conn) GroupAcceptInviteCode(code string) (jid string, err error) {
	request := []interface{}{"action", "invite", code}
	ch, err := wac.writeJson(request)
	if err != nil {
		return "", err
	}

	var response map[string]interface{}

	select {
	case r := <-ch:
		if err := json.Unmarshal([]byte(r), &response); err != nil {
			return "", fmt.Errorf("error decoding response message: %v\n", err)
		}
	case <-time.After(wac.msgTimeout):
		return "", fmt.Errorf("request timed out")
	}

	if int(response["status"].(float64)) != 200 {
		return "", fmt.Errorf("request responded with %d", response["status"])
	}

	return response["gid"].(string), nil
}
