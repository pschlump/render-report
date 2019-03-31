package main

// This file is MIT Licensed.

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/pschlump/Go-FTL/server/sizlib"
	"github.com/pschlump/MiscLib"
	"github.com/pschlump/godebug"
	"github.com/vjeantet/jodaTime"
)

/*

SEE: https://github.com/vjeantet/jodaTime

 Symbol  Meaning                      Presentation  Examples
 ------  -------                      ------------  -------
 G       era                          text          AD
 C       century of era (>=0)         number        20
 Y       year of era (>=0)            year          1996

 x       weekyear                     year          1996
 w       week of weekyear             number        27
 e       day of week                  number        2
 E       day of week                  text          Tuesday; Tue

 y       year                         year          1996
 D       day of year                  number        189
 M       month of year                month         July; Jul; 07
 d       day of month                 number        10

 a       halfday of day               text          PM
 K       hour of halfday (0~11)       number        0
 h       clockhour of halfday (1~12)  number        12

 H       hour of day (0~23)           number        0
 k       clockhour of day (1~24)      number        24
 m       minute of hour               number        30
 s       second of minute             number        55
 S       fraction of second           number        978

 z       time zone                    text          Pacific Standard Time; PST
 Z       time zone offset/id          zone          -0800; -08:00; America/Los_Angeles

 '       escape for text              delimiter
 ''      single quote                 literal       '
*/

type ReportQuery struct {
	Fmt string // row | table
	To  string // example "contact_information"
	Qry string // select ranch_name, real_name from t_reg_info where id = $1
}

type ReportSet struct {
	TemplateName string
	Queries      []ReportQuery
}

type ReportSetup struct {
	ReportFileName string      // File to output to
	ReportStatus   string      // Errors if any
	ReportData     []ReportSet // set of Templates to run with data before running template.
}

var R0 = ReportSetup{
	ReportFileName: "./tmpl/report.tmpl",
	ReportData: []ReportSet{
		{
			TemplateName: "header",
			Queries: []ReportQuery{
				{
					Fmt: "row",
					To:  ".",
					Qry: "select * from t_reg_info where id = $1",
				},
				{
					Fmt: "row",
					To:  ".",
					Qry: `select "email" as "primary_email" from "t_email" where "reg_info_id" = $1 and "address_type" = 'Primary' limit 1`,
				},
				{
					Fmt: "table",
					To:  "employee_resp",
					Qry: `select * from "t_employee_resp" where "reg_info_id" = $1 order by "seq"`,
				},
				{
					Fmt: "table",
					To:  "phone_no",
					Qry: `select * from "t_phone_no" where "reg_info_id" = $1 order by "created"`,
				},
				{
					Fmt: "table",
					To:  "physical_loc",
					Qry: `select * from "t_physical_loc" where "reg_info_id" = $1 order by "created"`,
				},
				//{
				//	Fmt: "table",
				//	To:  "address_type",
				//	Qry: `select * from "t_address_type" order by "created"`,
				//},
				{
					Fmt: "table",
					To:  "address",
					Qry: `select * from "t_address" where "reg_info_id" = $1 order by "created"`,
				},
				{
					Fmt: "table",
					To:  "program",
					Qry: `select * from "t_program" where "reg_info_id" = $1 order by "created"`,
				},
				{
					Fmt: "table",
					To:  "email",
					Qry: `select * from "t_email" where "reg_info_id" = $1 order by "created"`,
				},
				{
					Fmt: "table",
					To:  "ranch_locations",
					Qry: `select * from "t_ranch_locations" where "reg_info_id" = $1 order by "created"`,
				},
				{
					Fmt: "table",
					To:  "known_record_types",
					Qry: `select * from "t_known_record_type" where $1 = $1 order by "created"`,
				},
				{
					Fmt: "table",
					To:  "record_type",
					Qry: `select * from "t_record_type" where "reg_info_id" = $1 order by "created"`,
				},
				{
					Fmt: "table",
					To:  "calving_info",
					Qry: `select * from "t_calving_info" where "reg_info_id" = $1 order by "created"`,
				},
			},
		},
		{TemplateName: "sec_1"},
		{TemplateName: "sec_2"},
		{TemplateName: "sec_3"},
		{TemplateName: "footer"},
	},
}

// Given ID in t_reg_info, generate a HTML report for this registration.
func RenderReport(id string) (htmlOut string, err error) {
	var buffer bytes.Buffer

	g_data := make(map[string]interface{})
	for ii, rd := range R0.ReportData {

		sr := make([]string, 0, 5)
		nr := make([]int, 0, 5)
		// run all fo the queries and setup the data.
		for qn, q9 := range R0.ReportData {
			for qn1, qq := range q9.Queries {
				stmt := qq.Qry
				sr = append(sr, stmt)
				rows, err := SQLQuery(stmt, id)
				defer rows.Close()
				data, _, _ := sizlib.RowsToInterface(rows)
				if err != nil {
					fmt.Printf("Error: %s on %s with %s\n", err, stmt, id)
					fmt.Fprintf(os.Stderr, "%sError: %s on %s with %s%s\n", MiscLib.ColorRed, err, stmt, id, MiscLib.ColorReset)
					continue
				}
				nr = append(nr, len(data))
				if qq.Fmt == "row" {
					if qq.To == "." {
						if len(data) == 0 {
							// no data - that's ok.
						} else if len(data) == 1 {
							for key, val := range data[0] {
								g_data[key] = val
							}
						} else {
							fmt.Printf("Error: %s: %s: Got %d rows when expecting 1 row on 'row' specified query, data=%s\n",
								rd.TemplateName, len(data), stmt, godebug.SVar(data))
						}
					} else {
						tmp := make(map[string]interface{})
						if len(data) == 0 {
							// no data - that's ok.
						} else if len(data) == 1 {
							for key, val := range data[0] {
								tmp[key] = val
							}
						} else {
							fmt.Printf("Error: %s: %s: Got %d rows when expecting 1 row on 'row' specified query, data=%s\n",
								rd.TemplateName, len(data), stmt, godebug.SVar(data))
						}
						g_data[qq.To] = tmp
					}
				} else if qq.Fmt == "table" {
					g_data[qq.To] = make([]map[string]interface{}, 0, 1) // empty - no data
					if data == nil || len(data) == 0 {
						g_data[qq.To+"_length"] = 0
						g_data[qq.To+"_max_index"] = -1
					} else { // if len(data) >= 1 {
						g_data[qq.To] = data
						g_data[qq.To+"_length"] = len(data)
						g_data[qq.To+"_max_index"] = len(data) - 1
					}
				} else {
					fmt.Printf("Error: %s: qq.Fmt invalid [%s][%s][%v,%v] should be row/table.\n",
						R0.ReportFileName, rd.TemplateName, qq.Fmt, qn, qn1)
				}
			}
		}

		if db_flag["RenderReport"] {
			fmt.Printf("Report Data For Template=[%s]: %v : %s AT:%s\n", rd.TemplateName, ii, godebug.SVarI(g_data), godebug.LF())
		}

		g_data["__output_file_name__"] = R0.ReportFileName
		g_data["__template_name__"] = rd.TemplateName
		g_data["__stmt__"] = sr
		g_data["__nr__"] = nr
		g_data["__today_ISO__"] = jodaTime.Format("YYYY.MM.dd", time.Now())
		g_data["__today_US__"] = jodaTime.Format("dd/MMM/YYYY", time.Now())

		s := RunTemplate(R0.ReportFileName, rd.TemplateName, g_data)
		buffer.WriteString(s)
	}

	htmlOut = buffer.String()
	return
}
