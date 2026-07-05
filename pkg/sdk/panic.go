package sdk

func panicIfErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}
