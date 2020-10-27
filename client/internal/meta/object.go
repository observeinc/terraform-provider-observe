package meta

import (
	"fmt"
	"strconv"
)

type ObjectIdScalar int64

func (i ObjectIdScalar) Int64() int64 {
	return int64(i)
}

func (i ObjectIdScalar) String() string {
	return fmt.Sprintf("%d", i)
}

func ObjectIdScalarPointer(i int64) *ObjectIdScalar {
	if i <= 0 {
		return nil
	}
	o := ObjectIdScalar(i)
	return &o
}

func (o *ObjectIdScalar) UnmarshalJSON(b []byte) error {
	l := len(b)
	var str string
	if l < 3 || b[0] != '"' || b[l-1] != '"' {
		//	this is bad -- we should never marshal IDs as raw numbers!
		str = string(b)
	} else {
		str = string(b[1 : l-1])
	}
	oid, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return fmt.Errorf("JSON decode ObjectIdScalar %s failed: %s", b, err)
	}
	*o = ObjectIdScalar(oid)
	return nil
}

func (o ObjectIdScalar) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%d"`, int64(o))), nil
}
