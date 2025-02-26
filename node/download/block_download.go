package download

import (
	"context"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	titanRsa "github.com/linguohua/titan/node/rsa"

	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
	"github.com/linguohua/titan/api"
	"github.com/linguohua/titan/blockstore"
	"github.com/linguohua/titan/lib/limiter"
	"github.com/linguohua/titan/node/device"
	"github.com/linguohua/titan/node/helper"
	"github.com/linguohua/titan/node/validate"
	"golang.org/x/time/rate"
)

var log = logging.Logger("download")

type BlockDownload struct {
	limiter    *rate.Limiter
	blockStore blockstore.BlockStore
	publicKey  *rsa.PublicKey
	scheduler  api.Scheduler
	device     *device.Device
	validate   *validate.Validate
	srvAddr    string
}

func NewBlockDownload(limiter *rate.Limiter, params *helper.NodeParams, device *device.Device, validate *validate.Validate) *BlockDownload {
	var blockDownload = &BlockDownload{
		limiter:    limiter,
		blockStore: params.BlockStore,
		scheduler:  params.Scheduler,
		srvAddr:    params.DownloadSrvAddr,
		validate:   validate,
		device:     device}

	go blockDownload.startDownloadServer()

	return blockDownload
}

func (bd *BlockDownload) resultFailed(w http.ResponseWriter, r *http.Request, sn int64, sign []byte, err error) {
	log.Errorf("result failed:%s", err.Error())

	if sign != nil {
		result := api.NodeBlockDownloadResult{SN: sn, Sign: sign, DownloadSpeed: 0, BlockSize: 0, Result: false, FailedReason: err.Error()}
		go bd.downloadBlockResult(result)
	}

	if err == datastore.ErrNotFound {
		http.NotFound(w, r)
		return
	}

	http.Error(w, err.Error(), http.StatusBadRequest)
}

func (bd *BlockDownload) getBlock(w http.ResponseWriter, r *http.Request) {
	appName := r.Header.Get("App-Name")
	// sign := r.Header.Get("Sign")
	cidStr := r.URL.Query().Get("cid")
	signStr := r.URL.Query().Get("sign")
	snStr := r.URL.Query().Get("sn")
	signTime := r.URL.Query().Get("signTime")
	timeout := r.URL.Query().Get("timeout")

	log.Infof("GetBlock, App-Name:%s, sign:%s, sn:%s, signTime:%s, timeout:%s,  cid:%s", appName, signStr, snStr, signTime, timeout, cidStr)

	sn, err := strconv.ParseInt(snStr, 10, 64)
	if err != nil {
		bd.resultFailed(w, r, 0, nil, fmt.Errorf("Parser param sn(%s) error:%s", snStr, err.Error()))
		return
	}

	sign, err := hex.DecodeString(signStr)
	if err != nil {
		bd.resultFailed(w, r, 0, nil, fmt.Errorf("DecodeString sign(%s) error:%s", signStr, err.Error()))
		return
	}
	if bd.publicKey == nil {
		bd.resultFailed(w, r, sn, sign, fmt.Errorf("node %s publicKey == nil", bd.device.GetDeviceID()))
		return
	}

	content := cidStr + snStr + signTime + timeout
	err = titanRsa.VerifyRsaSign(bd.publicKey, sign, content)
	if err != nil {
		bd.resultFailed(w, r, sn, sign, fmt.Errorf("Verify sign cid:%s,sn:%s,signTime:%s, timeout:%s, error:%s,", cidStr, snStr, signTime, timeout, err.Error()))
		return
	}

	blockHash, err := helper.CIDString2HashString(cidStr)
	if err != nil {
		bd.resultFailed(w, r, sn, sign, fmt.Errorf("Parser param cid(%s) error:%s", cidStr, err.Error()))
		return
	}

	reader, err := bd.blockStore.GetReader(blockHash)
	if err != nil {
		bd.resultFailed(w, r, sn, sign, err)
		return
	}
	defer reader.Close()

	bd.validate.CancelValidate()

	contentDisposition := fmt.Sprintf("attachment; filename=%s", cidStr)
	w.Header().Set("Content-Disposition", contentDisposition)
	w.Header().Set("Content-Length", strconv.FormatInt(reader.Size(), 10))

	now := time.Now()

	n, err := io.Copy(w, limiter.NewReader(reader, bd.limiter))
	if err != nil {
		log.Errorf("GetBlock, io.Copy error:%v", err)
		return
	}

	costTime := time.Now().Sub(now)

	var speedRate = int64(0)
	if costTime != 0 {
		speedRate = int64(float64(n) / float64(costTime) * float64(time.Second))
	}

	result := api.NodeBlockDownloadResult{SN: sn, Sign: sign, DownloadSpeed: speedRate, BlockSize: int(n), Result: true}
	go bd.downloadBlockResult(result)

	log.Infof("Download block %s costTime %d, size %d, speed %d", cidStr, costTime, n, speedRate)

	return
}

func getClientIP(r *http.Request) string {
	reqIP := r.Header.Get("X-Real-IP")
	if reqIP == "" {
		h, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Errorf("could not get ip from: %s, err: %s", r.RemoteAddr, err)
		}
		reqIP = h
	}

	return reqIP
}

func (bd *BlockDownload) downloadBlockResult(result api.NodeBlockDownloadResult) {
	ctx, cancel := context.WithTimeout(context.Background(), helper.SchedulerApiTimeout*time.Second)
	defer cancel()

	err := bd.scheduler.NodeResultForUserDownloadBlock(ctx, result)
	if err != nil {
		log.Errorf("downloadBlockResult error:%s", err.Error())
	}
}

func (bd *BlockDownload) startDownloadServer() {
	mux := http.NewServeMux()
	mux.HandleFunc(helper.DownloadSrvPath, bd.getBlock)

	srv := &http.Server{
		Handler: mux,
		Addr:    bd.srvAddr,
	}

	nl, err := net.Listen("tcp", bd.srvAddr)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("download server listen on %s", bd.srvAddr)

	err = srv.Serve(nl)
	if err != nil {
		log.Fatal(err)
	}
}

// set download server upload speed
func (bd *BlockDownload) SetDownloadSpeed(ctx context.Context, speedRate int64) error {
	log.Infof("set download speed %d", speedRate)
	if bd.limiter == nil {
		return fmt.Errorf("edge.limiter == nil")
	}
	bd.limiter.SetLimit(rate.Limit(speedRate))
	bd.limiter.SetBurst(int(speedRate))
	bd.device.SetBandwidthUp(speedRate)
	return nil
}

func (bd *BlockDownload) UnlimitDownloadSpeed() error {
	log.Debug("UnlimitDownloadSpeed")
	if bd.limiter == nil {
		return fmt.Errorf("edge.limiter == nil")
	}

	bd.limiter.SetLimit(rate.Inf)
	bd.limiter.SetBurst(0)
	bd.device.SetBandwidthUp(int64(bd.limiter.Limit()))
	return nil
}

func (bd *BlockDownload) GetRateLimit() int64 {
	log.Debug("GenerateDownloadToken")
	return int64(bd.limiter.Limit())
}

func (bd *BlockDownload) GetDownloadSrvURL() string {
	addrSplit := strings.Split(bd.srvAddr, ":")
	url := fmt.Sprintf("http://%s:%s%s", bd.device.GetExternaIP(), addrSplit[1], helper.DownloadSrvPath)
	return url
}

func (bd *BlockDownload) LoadPublicKey() error {
	ctx, cancel := context.WithTimeout(context.Background(), helper.SchedulerApiTimeout*time.Second)
	defer cancel()

	publicKeyStr, err := bd.scheduler.GetPublicKey(ctx)
	if err != nil {
		return err
	}

	bd.publicKey, err = titanRsa.Pem2PublicKey(publicKeyStr)
	if err != nil {
		return err
	}
	return nil
}
