package controller

import (
	"gin-research-sys/internal/controller/req"
	"gin-research-sys/internal/controller/res"
	"gin-research-sys/internal/model"
	"gin-research-sys/internal/service"
	"github.com/360EntSecGroup-Skylar/excelize/v2"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"net/http"
	"strconv"
)

type IResearchController interface {
	List(c *gin.Context)
	Create(c *gin.Context)
	Retrieve(c *gin.Context)
	Update(c *gin.Context)
	Destroy(c *gin.Context)

	DownloadExcel(ctx *gin.Context)

	MgoRetrieve(ctx *gin.Context)
}
type ResearchController struct{}

func NewResearchController() IResearchController {
	return ResearchController{}
}

var researchServices = service.NewResearchService()
var researchMgoServices = service.NewResearchMgoService()

func (r ResearchController) List(ctx *gin.Context) {
	pg := req.PaginationQuery{}
	if err := ctx.ShouldBindQuery(&pg); err != nil {
		log.Println(err.Error())
		res.Success(ctx, nil, "query error")
		return
	}

	var researches []model.Research
	var total int64
	if err := researchServices.List(pg.Page, pg.Size, &researches, &total); err != nil {
		log.Println(err.Error())
		res.Success(ctx, nil, "list error")
	}
	res.Success(ctx, gin.H{
		"page":    pg.Page,
		"size":    pg.Size,
		"results": researches,
		"total":   total,
	}, "")
}

func (r ResearchController) Retrieve(ctx *gin.Context) {
	idString := ctx.Param("id")
	id, err := strconv.Atoi(idString)
	if err != nil {
		log.Println(err.Error())
		res.Fail(ctx, gin.H{}, "param is error")
		return
	}
	research := model.Research{}
	if err = researchServices.Retrieve(&research, id); err != nil {
		log.Println(err.Error())
		res.Fail(ctx, gin.H{}, "retrieve1 fail")
		return
	}
	researchMgo := model.ResearchMgo{}
	if err = researchMgoServices.Retrieve(&researchMgo, research.ResearchID); err != nil {
		log.Println(err.Error())
		res.Fail(ctx, gin.H{}, "retrieve2 fail")
		return
	}

	res.Success(ctx, gin.H{"research": gin.H{
		"id":          research.ID,
		"title":       research.Title,
		"desc":        research.Desc,
		"status":      research.Status,
		"once":        research.Once,
		"researchID":  research.ResearchID,
		"detail":      researchMgo.Detail,
		"rules":       researchMgo.Rules,
		"fieldsValue": researchMgo.FieldsValue,
	}}, "")
}

func (r ResearchController) Create(c *gin.Context) {
	createReq := req.ResearchCreateReq{}
	if err := c.ShouldBindJSON(&createReq); err != nil {
		res.Fail(c, gin.H{}, "payload error")
		return
	}
	// there needs mongo transaction
	researchMgo := model.ResearchMgo{
		Detail:      createReq.Detail,
		Rules:       createReq.Rules,
		FieldsValue: createReq.FieldsValue,
	}
	result, err := researchMgoServices.Create(&researchMgo)
	if err != nil {
		res.Fail(c, gin.H{}, "create error")
		return
	}
	claims := jwt.ExtractClaims(c)
	id := int(claims["id"].(float64))
	research := model.Research{
		Title:  createReq.Title,
		Desc:   createReq.Desc,
		Once:   createReq.Once,
		UserID: id,
	}
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		research.ResearchID = oid.Hex()
	} else {
		log.Println(ok)
		res.Fail(c, gin.H{}, "create fail")
		return
	}
	if err = researchServices.Create(&research); err != nil {
		log.Println(err.Error())
		res.Fail(c, gin.H{}, "create fail")
		return
	}
	res.Success(c, gin.H{}, "create success")
}

func (r ResearchController) Update(ctx *gin.Context) {
	idString := ctx.Param("id")
	id, err := strconv.Atoi(idString)
	if err != nil {
		log.Println(err.Error())
		res.Fail(ctx, gin.H{}, "param is error")
		return
	}
	updateReq := req.ResearchUpdateReq{}
	if err = ctx.ShouldBindJSON(&updateReq); err != nil {
		log.Println(err.Error())
		res.Fail(ctx, gin.H{}, "payload is error")
		return
	}
	research := model.Research{}
	if err = researchServices.Retrieve(&research, id); err != nil {
		log.Println(err.Error())
		res.Fail(ctx, gin.H{}, "retrieve fail")
		return
	}
	research.Title = updateReq.Title
	research.Desc = updateReq.Desc
	research.Once = updateReq.Once
	research.Status = updateReq.Status
	if err = researchServices.Update(&research); err != nil {
		log.Println(err.Error())
		res.Fail(ctx, gin.H{}, "update fail")
		return
	}
	//res.Success(ctx, gin.H{}, "update success")
	//research := bson.M{}
	//if err := researchServices.Update(idString, research); err != nil {
	//	res.Fail(ctx, gin.H{}, "update fail")
	//	log.Println(err.Error())
	//	return
	//}
	res.Success(ctx, gin.H{}, "update success")
}

func (r ResearchController) Destroy(ctx *gin.Context) {
	panic("implement me")
}

func (r ResearchController) DownloadExcel(ctx *gin.Context) {
	idString := ctx.Param("id")
	// get research
	researchMgo := model.ResearchMgo{}
	if err := researchMgoServices.Retrieve(&researchMgo, idString); err != nil {
		log.Println(err.Error())
		res.Fail(ctx, gin.H{}, "retrieve2 fail")
		return
	}
	// set the excel title line
	var fields []string
	titleRow := make([]interface{}, 0)
	titleRow = append(titleRow, "用户名", "IP地址", "填写时间")
	for _, v := range researchMgo.Detail {
		titleRow = append(titleRow, v["label"].(string))
		fields = append(fields, v["fieldId"].(string))
	}

	// get record list
	var records []model.Record
	var total int64
	if err := recordServices.ListID(idString, &records, &total); err != nil {
		log.Println(err.Error())
		res.Success(ctx, nil, "list error")
		return
	}

	// 1. start generate excel
	xlsx := excelize.NewFile()
	// 2. new StreamWriter
	streamWriter, err := xlsx.NewStreamWriter("Sheet1")
	if err != nil {
		println(err.Error())
	}
	if _, err = xlsx.NewStyle(`{"font":{"color":"#777777"}}`); err != nil {
		println(err.Error())
	}
	// 3. write title
	if err = streamWriter.SetRow("A1", titleRow); err != nil {
		return
	}

	// 4. write record data
	for k, v := range records {
		row := make([]interface{}, 0)
		row = append(row, v.User.Username)
		row = append(row, v.IP)
		row = append(row, v.CreatedAt.Format("2006-01-02 15:04:05"))
		for colID := 0; colID < len(fields); colID++ {
			mgo := model.RecordMgo{}
			if err = recordMgoServices.Retrieve(&mgo, v.RecordID); err != nil {
				println(err.Error())
			}
			row = append(row, mgo.FieldsValue[fields[colID]])
		}
		cell, _ := excelize.CoordinatesToCellName(1, k+2)
		if err = streamWriter.SetRow(cell, row); err != nil {
			println(err.Error())
		}
	}
	// 5. flush streamWriter
	if err = streamWriter.Flush(); err != nil {
		println(err.Error())
	}
	ctx.Header("response-type", "blob")
	data, _ := xlsx.WriteToBuffer()
	ctx.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data.Bytes())
}

func (r ResearchController) MgoRetrieve(ctx *gin.Context) {
	idString := ctx.Param("id")

	researchMgo := model.ResearchMgo{}
	if err := researchMgoServices.Retrieve(&researchMgo, idString); err != nil {
		log.Println(err.Error())
		res.Fail(ctx, gin.H{}, "retrieve2 fail")
		return
	}

	res.Success(ctx, gin.H{"research": gin.H{
		"detail":      researchMgo.Detail,
		"rules":       researchMgo.Rules,
		"fieldsValue": researchMgo.FieldsValue,
	}}, "")
}