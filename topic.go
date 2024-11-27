package siq

type Topic string

func NewTopic(topic string) Topic {
	return Topic(topic)
}
