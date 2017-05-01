package constraints

import (
	"testing"

	"github.com/stretchr/testify/suite"

	mesos "mesos/v1"
)

type LabelValuesTestSuite struct {
	suite.Suite
}

func (suite *LabelValuesTestSuite) TestGetHostLabelValues() {
	// An attribute of range type.
	rangeName := "range"
	rangeType := mesos.Value_RANGES
	rangeBegin := uint64(100)
	rangeEnd := uint64(200)
	ranges := mesos.Value_Ranges{
		Range: []*mesos.Value_Range{
			{
				Begin: &rangeBegin,
				End:   &rangeEnd,
			},
		},
	}
	rangeAttr := mesos.Attribute{
		Name:   &rangeName,
		Type:   &rangeType,
		Ranges: &ranges,
	}

	// An attribute of set type.
	setName := "set"
	setType := mesos.Value_SET
	setValues := mesos.Value_Set{
		Item: []string{"set_value1", "set_value2"},
	}
	setAttr := mesos.Attribute{
		Name: &setName,
		Type: &setType,
		Set:  &setValues,
	}

	// An attribute of scalar type.
	scalarName := "scalar"
	scalarType := mesos.Value_SCALAR
	sv := float64(1.0)
	scalarValue := mesos.Value_Scalar{
		Value: &sv,
	}
	scalarAttr := mesos.Attribute{
		Name:   &scalarName,
		Type:   &scalarType,
		Scalar: &scalarValue,
	}

	// An attribute of text type.
	textName := "text"
	textType := mesos.Value_TEXT
	tv := "text-value"
	textValue := mesos.Value_Text{
		Value: &tv,
	}
	textAttr := mesos.Attribute{
		Name: &textName,
		Type: &textType,
		Text: &textValue,
	}

	attributes := []*mesos.Attribute{
		&rangeAttr,
		&setAttr,
		&scalarAttr,
		&textAttr,
	}

	hostname := "test-host"

	res := GetHostLabelValues(hostname, attributes)
	// hostname, text, set and scalar.
	suite.Equal(4, len(res), "result: ", res)
	suite.Equal(map[string]uint32{hostname: 1}, res[HostNameKey])
	suite.Equal(
		map[string]uint32{
			"set_value1": 1,
			"set_value2": 1,
		},
		res[setName])
	suite.Equal(map[string]uint32{tv: 1}, res[textName])
	suite.Equal(map[string]uint32{"1.000000": 1}, res[scalarName])
}

func TestLabelValuesTestSuite(t *testing.T) {
	suite.Run(t, new(LabelValuesTestSuite))
}