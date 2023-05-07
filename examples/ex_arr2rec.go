package main

import (
	"fmt"
	gtr "github.com/hduplooy/gotransform"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Person struct {
	Name      string    `trf:"0;trim"`
	Surname   string    `trf:"1"`
	CallSign  string    `trf:"0,1;trimall;toupper;join='_'"`
	Age       int       `trf:"2"`
	DOB       time.Time `trf:"3"`          // ;format='2006-01-02'
	IPAddress []byte    `trf:"4;trans=IP"` // ;trim;rtrim;ltrim;trimall;tolower;toupper;capitalize;fill='0';lalign;ralign;center;width=10;pos=20;currency='$;dec=true;decimals=4';
	Signature []byte    `trf:"5;base64"`
	// for byte fields   ;base64;hex;
}

func init() {
	fmt.Println("Main init")
	gtr.AddTransformer("IP", func(fld reflect.Value, value string) error {
		fmt.Println("Cehcking " + value)
		vals := strings.Split(value, ".")
		res := make([]byte, 4)
		for i, val := range vals {
			tmp, _ := strconv.Atoi(val)
			res[i] = byte(tmp)
		}
		fld.Set(reflect.ValueOf(res))
		return nil
	}, func(fld reflect.Value) (string, error) {
		ip := fld.Interface().([]byte)
		return fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3]), nil
	})
}

func (p *Person) String() string {
	return fmt.Sprintf("Person(Name: %s, Surname: %s, Age: %d, DOB: %v)", p.Name, p.Surname, p.Age, p.DOB)
}

func setIP(fld reflect.Value, value string) error {
	fmt.Println("Cehcking " + value)
	vals := strings.Split(value, ".")
	res := make([]byte, 4)
	for i, val := range vals {
		tmp, _ := strconv.Atoi(val)
		res[i] = byte(tmp)
	}
	fld.Set(reflect.ValueOf(res))
	return nil
}

func main() {
	tr, err := gtr.NewTransform(Person{})
	if err != nil {
		panic(err)
	}
	//tr.SetSetter("IPAddress", setIP)

	values := [][]string{
		{"      John Doby     ", "Doe", "23", "2001-01-01", "1.2.3.4", "MDEyMzQ="},
		{"Peter Jack", "Pan", "13", "1810-02-02", "2.1.4.3", ""},
	}

	tmp, err := tr.ArrToStruct(values)
	if err != nil {
		panic(err)
	}
	persons := tmp.([]Person)
	for _, person := range persons {
		fmt.Println(person)
	}

}
