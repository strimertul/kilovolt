package kv

import (
	"bytes"
)

type subscriptionManager struct {
	keySubscribers    map[string][]int64
	prefixSubscribers map[string][]int64
	hub               *Hub
}

func makeSubscriptionManager() *subscriptionManager {
	return &subscriptionManager{
		keySubscribers:    make(map[string][]int64),
		prefixSubscribers: make(map[string][]int64),
	}
}

func (s *subscriptionManager) SubscribeKey(uid int64, key string) {
	s.keySubscribers[key] = append(s.keySubscribers[key], uid)
}

func (s *subscriptionManager) SubscribePrefix(uid int64, prefix string) {
	s.prefixSubscribers[prefix] = append(s.prefixSubscribers[prefix], uid)
}

func (s *subscriptionManager) UnsubscribeKey(uid int64, key string) {
	subscribers := s.keySubscribers[key]
	for i, subscriber := range subscribers {
		if subscriber == uid {
			s.keySubscribers[key] = append(subscribers[:i], subscribers[i+1:]...)
			break
		}
	}
}

func (s *subscriptionManager) UnsubscribePrefix(uid int64, prefix string) {
	subscribers := s.prefixSubscribers[prefix]
	for i, subscriber := range subscribers {
		if subscriber == uid {
			s.prefixSubscribers[prefix] = append(subscribers[:i], subscribers[i+1:]...)
			break
		}
	}
}

func (s *subscriptionManager) UnsubscribeAll(uid int64) {
	for key, subscribers := range s.keySubscribers {
		for i, subscriber := range subscribers {
			if subscriber == uid {
				s.keySubscribers[key] = append(subscribers[:i], subscribers[i+1:]...)
				break
			}
		}
	}

	for prefix, subscribers := range s.prefixSubscribers {
		for i, subscriber := range subscribers {
			if subscriber == uid {
				s.prefixSubscribers[prefix] = append(subscribers[:i], subscribers[i+1:]...)
				break
			}
		}
	}
}

func (s *subscriptionManager) GetSubscribers(key string) []int64 {
	subscribers := make(map[int64]bool)

	// Get subscribers for key
	if keySubscribers, ok := s.keySubscribers[key]; ok {
		for _, subscriber := range keySubscribers {
			subscribers[subscriber] = true
		}
	}

	// Get subscribers for prefix
	for prefix, prefixSubscribers := range s.prefixSubscribers {
		if bytes.HasPrefix([]byte(key), []byte(prefix)) {
			for _, subscriber := range prefixSubscribers {
				subscribers[subscriber] = true
			}
		}
	}

	// Convert to array
	result := []int64{}
	for subscriber := range subscribers {
		result = append(result, subscriber)
	}

	return result
}

func (s *subscriptionManager) KeyChanged(key string, value string) {
	// Notify subscribers
	clients := s.GetSubscribers(key)
	for _, clientID := range clients {
		client, ok := s.hub.clients.GetByID(clientID)
		if ok {
			options := client.Options()
			msg, _ := json.Marshal(Push{"push", key[len(options.Namespace):], value})
			client.SendMessage(msg)
		}
	}
}
