package api

import (
	"context"
)

type Locator interface {
	Common
	GetAccessPoints(ctx context.Context, deviceID string) ([]string, error)                                                 //perm:read
	AddAccessPoints(ctx context.Context, areaID string, schedulerURL string, weight int, schedulerAccessToken string) error //perm:admin
	RemoveAccessPoints(ctx context.Context, areaID string) error                                                            //perm:admin                                  //perm:admin
	ListAccessPoints(ctx context.Context) (areaIDs []string, err error)                                                     //perm:admin
	ShowAccessPoint(ctx context.Context, areaID string) (AccessPoint, error)                                                //perm:admin

	DeviceOnline(ctx context.Context, deviceID string, areaID string, port int) error //perm:write
	DeviceOffline(ctx context.Context, deviceID string) error                         //perm:write

	// user api
	GetDownloadInfosWithBlocks(ctx context.Context, cids []string, publicKey string) (map[string][]DownloadInfoResult, error) //perm:read
	GetDownloadInfoWithBlocks(ctx context.Context, cids []string, publicKey string) (map[string]DownloadInfoResult, error)    //perm:read
	GetDownloadInfoWithBlock(ctx context.Context, cids string, publicKey string) (DownloadInfoResult, error)                  //perm:read
	// user send result when user download block complete
	UserDownloadBlockResults(ctx context.Context, results []UserBlockDownloadResult) error //perm:read

	RegisterNode(ctx context.Context, areaID, schedulerURL string, nodeType NodeType, count int) ([]NodeRegisterInfo, error) // perm:admin
	LoadAccessPointsForWeb(ctx context.Context) ([]AccessPoint, error)                                                       // perm:admin
	LoadUserAccessPoint(ctx context.Context, userIP string) (AccessPoint, error)                                             // perm:admin
}

type SchedulerInfo struct {
	URL    string
	Weight int
	// Online      bool
	// AccessToken string
}
type AccessPoint struct {
	AreaID         string
	SchedulerInfos []SchedulerInfo
}
