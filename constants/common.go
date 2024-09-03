package constants

const (
	// AmazonAPITokenURL is the URL used to fetch amazon access tokens
	AmazonAPITokenURL = "https://api.amazon.com/auth/token"

	// EndpointURL is the URL used to fetch the endpoints to use for all the other REST API call.
	// https://developer.amazon.com/docs/amazon-drive/ad-restful-api-account.html
	AmazonDriveEndpointURL = "https://drive.amazonaws.com/drive/v1/account/endpoint"

	// AMZClientOwnerName is the owner/client name of the Amazon Photos application.
	AMZClientOwnerName = "AMZClient"

	// CloudDriveWebOwnerName is the owner/client name of the Cloud Drive Web interface.
	//CloudDriveWebOwnerName = "CloudDriveWeb"
)
