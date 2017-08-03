package main

import "sort"

type sender struct {
	Name  string
	Count int
}

type senderList []sender

func (p senderList) Len() int           { return len(p) }
func (p senderList) Less(i, j int) bool { return p[i].Count < p[j].Count }
func (p senderList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func rank(senders map[string]int) senderList {
	sl := make(senderList, len(senders))
	i := 0
	for k, v := range senders {
		sl[i] = sender{k, v}
		i++
	}
	sort.Sort(sort.Reverse(sl))
	return sl
}
