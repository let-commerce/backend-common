package main

import (
	"github.com/let-commerce/backend-common/env"
	"github.com/speps/go-hashids"
)

var (
	salt = env.GetEnvVar("SALT")
	//prime      = env.GetEnvVar("PRIME")
	//modInverse = env.GetEnvVar("MOD_INVERSE")
	//random     = env.GetEnvVar("PRIME")
)

func EncodeHashId(id int) string {
	// read here: https://hashids.org/go/
	hd := hashids.NewData()
	hd.Salt = salt
	hd.MinLength = 6
	h, _ := hashids.NewWithData(hd)
	result, _ := h.Encode([]int{id})
	return result
}

func DecodeHashId(str string) int {
	hd := hashids.NewData()
	hd.Salt = salt
	hd.MinLength = 6
	h, _ := hashids.NewWithData(hd)
	numbers, _ := h.DecodeWithError(str)
	return numbers[0]
}

//func EncodeOptimus(id uint64) uint64 {
//	// read here: https://github.com/pjebs/optimus-go
//	o := optimus.New(prime, modInverse, random)
//	return o.Encode(id)
//}
//
//func DecodeOptimus(encodedId uint64) uint64 {
//	o := optimus.New(prime, modInverse, random)
//	return o.Decode(encodedId)
//}
