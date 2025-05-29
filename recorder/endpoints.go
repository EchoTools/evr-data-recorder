package recorder

var (
	EndpointSession = func(baseURL string) string {
		return baseURL + "/session"
	}

	EndpointPlayerBones = func(baseURL string) string {
		return baseURL + "/player_bones"
	}
)
