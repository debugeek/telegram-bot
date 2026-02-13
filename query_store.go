package tgbot

import (
	"strconv"
	"strings"
	"sync"
)

type queryStore[BOTDATA any, USERDATA any] struct {
	mu sync.Mutex

	maxPerSession  int
	bySession      map[int64]*LRUMap[string, pendingQuery[BOTDATA, USERDATA]]
	queryToSession map[string]int64
	querySeq       uint64
}

func newQueryStore[BOTDATA any, USERDATA any](maxPerSession int) *queryStore[BOTDATA, USERDATA] {
	if maxPerSession <= 0 {
		maxPerSession = 1
	}
	return &queryStore[BOTDATA, USERDATA]{
		maxPerSession:  maxPerSession,
		bySession:      make(map[int64]*LRUMap[string, pendingQuery[BOTDATA, USERDATA]]),
		queryToSession: make(map[string]int64),
	}
}

func (s *queryStore[BOTDATA, USERDATA]) Create(
	sessionID int64,
	options []string,
	handler func(*Session[BOTDATA, USERDATA], string),
) *InlineKeyboardMarkup {
	if len(options) == 0 || handler == nil {
		return nil
	}

	s.mu.Lock()
	queryID := strconv.FormatUint(s.querySeq+1, 10)
	s.querySeq++

	answers := make(map[string]string, len(options))
	keyboardRows := make([][]InlineKeyboardButton, 0, len(options))
	for idx, option := range options {
		key := strconv.Itoa(idx)
		callbackToken := strings.Join([]string{"q", queryID, key}, ":")
		keyboardRows = append(keyboardRows, []InlineKeyboardButton{
			{Text: option, CallbackData: callbackToken},
		})
		answers[key] = option
	}

	if existingSessionID, ok := s.queryToSession[queryID]; ok {
		if m := s.bySession[existingSessionID]; m != nil {
			m.Remove(queryID)
			if m.Len() == 0 {
				delete(s.bySession, existingSessionID)
			}
		}
		delete(s.queryToSession, queryID)
	}

	item := pendingQuery[BOTDATA, USERDATA]{
		sessionID: sessionID,
		answers:   answers,
		handler:   handler,
	}

	sessionMap := s.bySession[sessionID]
	if sessionMap == nil {
		sessionMap = NewLRUMap[string, pendingQuery[BOTDATA, USERDATA]](s.maxPerSession)
		s.bySession[sessionID] = sessionMap
	}

	evicted, evictedQueryID, _ := sessionMap.Put(queryID, item)
	s.queryToSession[queryID] = sessionID
	if evicted {
		delete(s.queryToSession, evictedQueryID)
	}
	s.mu.Unlock()

	return &InlineKeyboardMarkup{InlineKeyboard: keyboardRows}
}

func (s *queryStore[BOTDATA, USERDATA]) Take(queryID string) (pendingQuery[BOTDATA, USERDATA], bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID, ok := s.queryToSession[queryID]
	if !ok {
		var zero pendingQuery[BOTDATA, USERDATA]
		return zero, false
	}

	sessionMap := s.bySession[sessionID]
	if sessionMap == nil {
		var zero pendingQuery[BOTDATA, USERDATA]
		delete(s.queryToSession, queryID)
		return zero, false
	}

	item, ok := sessionMap.Take(queryID)
	if !ok {
		var zero pendingQuery[BOTDATA, USERDATA]
		delete(s.queryToSession, queryID)
		return zero, false
	}

	delete(s.queryToSession, queryID)
	if sessionMap.Len() == 0 {
		delete(s.bySession, sessionID)
	}
	return item, true
}
