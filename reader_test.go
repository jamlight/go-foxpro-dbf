package dbf

import (
	"fmt"
	"strings"
	"testing"
)

const (
	TEST_DBF_PATH  = "./testdbf/TEST.DBF"
	BENCH_DBF_PATH = "./testdbf/TEST.DBF" //For real benchmarks replace this with the path to a large DBF/FPT combo
)

var test_dbf *DBF

func TestOpenFile(t *testing.T) {
	var err error
	test_dbf, err = OpenFile(TEST_DBF_PATH)
	if err != nil {
		t.Fatal(err)
	}
}

//Quick check if the first field matches
func TestFieldHeader(t *testing.T) {
	want := "{Name:[73 68 0 0 0 0 0 0 0 0 0] Type:73 Pos:1 Len:4 Decimals:0 Flags:0 Next:5 Step:1 Reserved:[0 0 0 0 0 0 0 78]}"
	have := fmt.Sprintf("%+v", test_dbf.fields[0])
	if have != want {
		t.Errorf("First field from header does not match signature: Want %s, have %s", want, have)
	}
}

//Test if the modified date of Stat() matches the header
//This is therefore also a header test, these dates should be equal, but not sure if this is always true on every OS
//Update: Disable for now, fails on other timezones
/*
func TestStat(t *testing.T) {
	stat, err := test_dbf.Stat()
	if err != nil {
		t.Fatal(err)
	}
	stat_mod := stat.ModTime()
	hdr_mod := test_dbf.header.Modified()
	format := "20060102"
	if stat_mod.Format(format) != hdr_mod.Format(format) {
		t.Errorf("Modified date in header (%s) not equal to modified date in OS (%s)", hdr_mod.Format(format), stat_mod.Format(format))
	}
}*/
//test with size instead
func TestStatAndFileSize(t *testing.T) {
	stat, err := test_dbf.Stat()
	if err != nil {
		t.Fatal(err)
	}
	stat_size := stat.Size()
	hdr_size := test_dbf.header.FileSize()
	if stat_size != hdr_size {
		t.Errorf("Calculated header size: %d, stat size: %d", hdr_size, stat_size)
	}
}

//Tests if field headers have been parsed, fails if there are no fields
func TestFieldNames(t *testing.T) {
	fieldnames := test_dbf.FieldNames()
	want := 13
	if len(fieldnames) != want {
		t.Errorf("Expected %d fields, have %d", want, len(fieldnames))
	}
	//t.Log(fieldnames)
}

func TestNumFields(t *testing.T) {
	header := test_dbf.NumFields()
	header_calc := test_dbf.Header().NumFields()
	if header != header_calc {
		t.Errorf("NumFields not equal. DBF NumFields: %d, DBF Header NumField: %d", header, header_calc)
	}
}

func TestGoTo(t *testing.T) {
	err := test_dbf.GoTo(0)
	if err != nil {
		t.Error(err)
	}
	if !test_dbf.BOF() {
		t.Error("Expected to be at BOF")
	}
	err = test_dbf.GoTo(1)
	if err != nil {
		t.Error(err)
	}
	if test_dbf.EOF() {
		t.Error("Did not expect to be at EOF")
	}
	err = test_dbf.GoTo(4)
	if err != nil {
		if err != ErrEOF {
			t.Error(err)
		}
	}
	if !test_dbf.EOF() {
		t.Error("Expected to be at EOF")
	}
}

func TestSkip(t *testing.T) {
	test_dbf.GoTo(0)

	err := test_dbf.Skip(1)
	if err != nil {
		t.Error(err)
	}
	if test_dbf.EOF() {
		t.Error("Did not expect to be at EOF")
	}
	err = test_dbf.Skip(3)
	if err != nil {
		if err != ErrEOF {
			t.Error(err)
		}
	}
	if !test_dbf.EOF() {
		t.Error("Expected to be at EOF")
	}
	err = test_dbf.Skip(-20)
	if err != nil {
		if err != ErrBOF {
			t.Error(err)
		}
	}
	if !test_dbf.BOF() {
		t.Error("Expected to be at BOF")
	}
}

var want_values = []struct {
	pos                   int
	name, strval, strtype string
}{
	{0, "ID", "2", "int32"},
	{10, "NUMBER", "1.2345678999e+08", "float64"},
	{12, "BOOL", "true", "bool"},
	{2, "DATUM", "2015-02-03 00:00:00 +0000 UTC", "time.Time"},
	{7, "COMP_NAME", "TEST2", "string"},
}

func TestFieldPos(t *testing.T) {

	for _, want := range want_values {
		pos := test_dbf.FieldPos(want.name)
		if pos != want.pos {
			t.Errorf("Wanted fieldpos %d for field %s, have pos %d", want.pos, want.name, pos)
		}
	}
}

//Tests a complete record read, reads the second record which is also deleted,
//also tests getting field values from record object
func TestRecord(t *testing.T) {
	err := test_dbf.GoTo(1)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := test_dbf.Record()
	if err != nil {
		t.Fatal(err)
	}
	//t.Log(rec.data)

	//Get fields by pos
	for _, want := range want_values {
		val, err := rec.Field(want.pos)
		if err != nil {
			t.Error(err)
		}
		strval := strings.TrimSpace(fmt.Sprintf("%v", val))
		strtype := fmt.Sprintf("%T", val)

		if want.strval != strval || want.strtype != strtype {
			t.Errorf("Wanted value %s with type %s, have value %s with type %s", want.strval, want.strtype, strval, strtype)
		}
	}
}

//Test reading fields field by field
func TestField(t *testing.T) {
	for _, want := range want_values {
		val, err := test_dbf.Field(want.pos)
		if err != nil {
			t.Error(err)
		}
		strval := strings.TrimSpace(fmt.Sprintf("%v", val))
		strtype := fmt.Sprintf("%T", val)

		if want.strval != strval || want.strtype != strtype {
			t.Errorf("Wanted value %s with type %s, have value %s with type %s", want.strval, want.strtype, strval, strtype)
		}
	}
}

//Close file handles
func TestClose(t *testing.T) {
	err := test_dbf.Close()
	if err != nil {
		t.Fatal(err)
	}
}

//Benchmark for reading all records sequentially
//Use a large DBF/FPT combo for more realistic results
func BenchmarkReadRecords(b *testing.B) {
	for n := 0; n < b.N; n++ {
		err := func() error {
			dbf, err := OpenFile(BENCH_DBF_PATH)
			if err != nil {
				return err
			}
			defer dbf.Close()
			for i := uint32(0); i < dbf.NumRecords(); i++ {
				_, err := dbf.Record()
				if err != nil {
					return err
				}
				dbf.Skip(1)
			}
			return nil
		}()
		if err != nil {
			b.Fatal(err)
		}
	}
}
