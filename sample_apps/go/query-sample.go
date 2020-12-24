package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/timestreamquery"

	"flag"
	"fmt"
	"os"
)

func main() {
	// setup the query client
	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})

	if err != nil {
		fmt.Println("Error:")
		fmt.Println(err)
		return
	}

	// querySvc := timestreamquery.New(sess, &aws.Config{Endpoint: aws.String("query-cell0.timestream.us-east-1.amazonaws.com")})
	querySvc := timestreamquery.New(sess)

	queryStr := fmt.Sprintf(`
SELECT * FROM "devops"."host_metrics" ORDER BY time DESC LIMIT 10
	`)

	// process command line arguments
	queryPtr := flag.String("query", queryStr, "query string")
	filePtr := flag.String("outputfile", "", "output results file in the current folder")
	flag.Parse()
	var f *os.File
	if *filePtr != "" {
		var ferr error
		f, ferr = os.Create(*filePtr)
		check(ferr)
		defer f.Close()
	}

	runQuery(queryPtr, querySvc, f)
}

func fail(s string) {
	panic(s)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func write(f *os.File, s string) {
	if f != nil {
		_, err := f.WriteString(s + "\n")
		check(err)
		f.Sync()
	}
}

func processScalarType(data *timestreamquery.Datum) string {
	return *data.ScalarValue
}

func processTimeSeriesType(data []*timestreamquery.TimeSeriesDataPoint, columnInfo *timestreamquery.ColumnInfo) string {
	value := ""
	for k := 0; k < len(data); k++ {
		time := data[k].Time
		value += *time + ":"
		if columnInfo.Type.ScalarType != nil {
			value += processScalarType(data[k].Value)
		} else if columnInfo.Type.ArrayColumnInfo != nil {
			value += processArrayType(data[k].Value.ArrayValue, columnInfo.Type.ArrayColumnInfo)
		} else if columnInfo.Type.RowColumnInfo != nil {
			value += processRowType(data[k].Value.RowValue.Data, columnInfo.Type.RowColumnInfo)
		} else {
			fail("Bad data type")
		}
		if k != len(data)-1 {
			value += ", "
		}
	}
	return value
}

func processArrayType(datumList []*timestreamquery.Datum, columnInfo *timestreamquery.ColumnInfo) string {
	value := ""
	for k := 0; k < len(datumList); k++ {
		if columnInfo.Type.ScalarType != nil {
			value += processScalarType(datumList[k])
		} else if columnInfo.Type.TimeSeriesMeasureValueColumnInfo != nil {
			value += processTimeSeriesType(datumList[k].TimeSeriesValue, columnInfo.Type.TimeSeriesMeasureValueColumnInfo)
		} else if columnInfo.Type.ArrayColumnInfo != nil {
			value += "["
			value += processArrayType(datumList[k].ArrayValue, columnInfo.Type.ArrayColumnInfo)
			value += "]"
		} else if columnInfo.Type.RowColumnInfo != nil {
			value += "["
			value += processRowType(datumList[k].RowValue.Data, columnInfo.Type.RowColumnInfo)
			value += "]"
		} else {
			fail("Bad column type")
		}

		if k != len(datumList)-1 {
			value += ", "
		}
	}
	return value
}

func processRowType(data []*timestreamquery.Datum, metadata []*timestreamquery.ColumnInfo) string {
	value := ""
	for j := 0; j < len(data); j++ {
		if metadata[j].Type.ScalarType != nil {
			// process simple data types
			value += processScalarType(data[j])
		} else if metadata[j].Type.TimeSeriesMeasureValueColumnInfo != nil {
			// fmt.Println("Timeseries measure value column info")
			// fmt.Println(metadata[j].Type.TimeSeriesMeasureValueColumnInfo.Type)
			datapointList := data[j].TimeSeriesValue
			value += "["
			value += processTimeSeriesType(datapointList, metadata[j].Type.TimeSeriesMeasureValueColumnInfo)
			value += "]"
		} else if metadata[j].Type.ArrayColumnInfo != nil {
			columnInfo := metadata[j].Type.ArrayColumnInfo
			// fmt.Println("Array column info")
			// fmt.Println(columnInfo)
			datumList := data[j].ArrayValue
			value += "["
			value += processArrayType(datumList, columnInfo)
			value += "]"
		} else if metadata[j].Type.RowColumnInfo != nil {
			columnInfo := metadata[j].Type.RowColumnInfo
			datumList := data[j].RowValue.Data
			value += "["
			value += processRowType(datumList, columnInfo)
			value += "]"
		} else {
			panic("Bad column type")
		}
		// comma seperated column values
		if j != len(data)-1 {
			value += ", "
		}
	}
	return value
}

func runQuery(queryPtr *string, querySvc *timestreamquery.TimestreamQuery, f *os.File) {
	queryInput := &timestreamquery.QueryInput{
		QueryString: aws.String(*queryPtr),
	}
	fmt.Println("QueryInput:")
	fmt.Println(queryInput)
	// execute the query
	err := querySvc.QueryPages(queryInput,
		func(page *timestreamquery.QueryOutput, lastPage bool) bool {
			// process query response
			queryStatus := page.QueryStatus
			fmt.Println("Current query status:")
			fmt.Println(queryStatus)
			// query response metadata
			// includes column names and types
			metadata := page.ColumnInfo
			fmt.Println("Metadata:")
			fmt.Println(metadata)
			header := ""
			for i := 0; i < len(metadata); i++ {
				header += *metadata[i].Name
				if i != len(metadata)-1 {
					header += ", "
				}
			}
			write(f, header)

			// query response data
			fmt.Println("Data:")
			fmt.Println(page.Rows)
			// process rows
			rows := page.Rows
			for i := 0; i < len(rows); i++ {
				data := rows[i].Data
				value := processRowType(data, metadata)
				fmt.Println(value)
				write(f, value)
			}
			fmt.Println("Number of rows:", len(page.Rows))
			return true
		})
	if err != nil {
		fmt.Println("Error:")
		fmt.Println(err)
	}
}
