package credential

func targetName(service, account string) string {
	return service + ":" + account
}
