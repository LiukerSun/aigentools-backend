package epay

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type EpayDriver struct {
	GatewayURL string
	PID        string
	Key        string
}

func NewEpayDriver() *EpayDriver {
	return &EpayDriver{}
}

func (d *EpayDriver) SetConfig(config map[string]interface{}) error {
	if val, ok := config["url"].(string); ok {
		// Ensure URL ends with submit.php or handle logic if it's just base URL
		// Assuming user provides base URL like https://epay.com/
		baseURL := strings.TrimRight(val, "/")
		if !strings.HasSuffix(baseURL, "submit.php") {
			d.GatewayURL = baseURL + "/submit.php"
		} else {
			d.GatewayURL = baseURL
		}
	} else {
		return errors.New("missing url in config")
	}

	if val, ok := config["pid"].(string); ok {
		d.PID = val
	} else if val, ok := config["pid"].(float64); ok {
		d.PID = fmt.Sprintf("%.0f", val)
	} else {
		return errors.New("missing pid in config")
	}

	if val, ok := config["key"].(string); ok {
		d.Key = val
	} else {
		return errors.New("missing key in config")
	}
	return nil
}

func (d *EpayDriver) Pay(orderID string, amount float64, notifyURL string, returnURL string, params map[string]interface{}) (string, error) {
	// Construct params
	data := map[string]string{
		"pid":          d.PID,
		"type":         "alipay", // Default
		"out_trade_no": orderID,
		"notify_url":   notifyURL,
		"return_url":   returnURL,
		"name":         "Topup " + orderID,
		"money":        fmt.Sprintf("%.2f", amount),
	}

	if val, ok := params["type"].(string); ok {
		data["type"] = val
	}
	if val, ok := params["name"].(string); ok {
		data["name"] = val
	}

	sign := d.generateSign(data)
	data["sign"] = sign
	data["sign_type"] = "MD5"

	// Construct Query String
	q := url.Values{}
	for k, v := range data {
		q.Set(k, v)
	}

	return d.GatewayURL + "?" + q.Encode(), nil
}

func (d *EpayDriver) Notify(params map[string]interface{}) (bool, string, string, error) {
	// Verify Sign
	// Convert params to map[string]string
	data := make(map[string]string)
	var remoteSign string
	var orderID string
	var externalID string

	for k, v := range params {
		valStr := fmt.Sprintf("%v", v)
		if k == "sign" {
			remoteSign = valStr
			continue
		}
		if k == "sign_type" {
			continue
		}
		data[k] = valStr
		if k == "out_trade_no" {
			orderID = valStr
		}
		if k == "trade_no" {
			externalID = valStr
		}
	}

	localSign := d.generateSign(data)
	if localSign == remoteSign {
		return true, orderID, externalID, nil
	}
	return false, orderID, externalID, errors.New("signature mismatch")
}

func (d *EpayDriver) generateSign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, k := range keys {
		v := params[k]
		if v == "" || k == "sign" || k == "sign_type" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("&")
		}
		builder.WriteString(k)
		builder.WriteString("=")
		builder.WriteString(v)
	}
	builder.WriteString(d.Key)

	hash := md5.Sum([]byte(builder.String()))
	return hex.EncodeToString(hash[:])
}
