package util

import (
	"fmt"
	"strconv"
	"strings"
)

/*toCent  转换金额(单位：元)为金额(单位：分)，最多支持2位小数，如果超出则忽略多余小数
参数:
*	amount	  string    金额字符串，如100,000.01
返回值:
*	value	  int64     金额(单位:分)
*	err  	  error     ParseErr
*/
func ToCent(amount string) (int64, error) {
	val := strings.ReplaceAll(amount, ",", "")
	idx := strings.LastIndex(val, ".") //小数点位置
	n := len(val)

	switch n - idx - 1 {
	case n, 0: // 不存在，整数,单位为元，倍数为100
		val += "00"
	case 1:
		val += "0"
	default:
		//>=2位小数
		val = val[:idx+3]
	}

	val = strings.ReplaceAll(val, ".", "")
	if v, err := strconv.ParseInt(val, 10, 64); err == nil {
		return v, nil
	} else {
		return 0, fmt.Errorf("parse amount '%v' err->%w", amount, err)
	}

}
