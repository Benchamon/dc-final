package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	//"strconv"
	"github.com/dgrijalva/jwt-go"
	"strings"
	"encoding/base64"
	"net/http"
	"time"
	"path/filepath"
	"path"
	"os"
	"github.com/Benchamon/dc-final/scheduler"
	"github.com/Benchamon/dc-final/controller"
)

var Jobs = make(chan scheduler.Job)
var cantTests int

type User struct{
	user string
	passw string
	token string
}

var Users = make(map[string]User)

/*func imgSize(size int64) (string, ){
	var KB, MB, Max float64 = 1024, 1048576, 10485760
	FloatS := float64(size)
	if FloatS < KB{
		return strconv.FormatFloat(FloatS, 'f', 2, 64) + "b"
	} else if FloatS >= KB && FloatS < MB{
		return strconv.FormatFloat(FloatS/KB, 'f', 2, 64) + "Kb"
	} else if FloatS >= MB && FloatS <= Max{
		return strconv.FormatFloat(FloatS/MB, 'f', 2, 64) + "Mb"
	} else{
		return ""
	}
}*/

func CreateToken(user string) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["user"] = user
	claims["exp"] = time.Now().Add(time.Hour * 3).Unix()
	t, err := token.SignedString([]byte("our-secret"))
	claims["token"] = t
	return t, err
}

func login(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "application/json")
	fmt.Println("Response Type:", c.Writer.Header().Get("Content-Type"))
	params := strings.Split(c.Request.Header.Get("Authorization"), " ")

	auth, _ := base64.StdEncoding.DecodeString(params[1])
	fmt.Printf("User: %v\n", string(auth))

	userInfo := strings.Split(string(auth), ":")
	exists := false

	for _,u := range Users {
		if u.user == userInfo[0] {
			exists = true
		}
	}

	if !exists {
		newToken, err := CreateToken(userInfo[0] + "." + userInfo[1])
		if err != nil {
			c.JSON(http.StatusConflict, Rerror("Token not created"))
			return
		}
		newUser := User{
			user: userInfo[0],
			passw: userInfo[1],
			token: newToken,
		}
		Users[newToken] = newUser
		c.JSON(http.StatusOK, Rlogin(newUser.user, newUser.token))
	}
	if exists {
		c.JSON(http.StatusOK, Rerror("User already logged"))
	}
}

func logout(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "application/json")
	fmt.Println("Response Type:", c.Writer.Header().Get("Content-Type"))

	params := strings.Split(c.Request.Header.Get("Authorization"), " ")
	token := params[1]

	if _, ok := Users[token]; ok {
		c.JSON(http.StatusOK, RLogout(Users[token].user))
		delete(Users, token)
	} else {
		c.JSON(http.StatusConflict, Rerror("Token not valid"))
	}
}

func status(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "application/json")
	fmt.Println("Response Type:", c.Writer.Header().Get("Content-Type"))

	params := strings.Split(c.Request.Header.Get("Authorization"), " ")
	token := params[1]

	if _, ok := Users[token]; ok {
		c.JSON(http.StatusOK, Rstatus())
	} else {
		c.JSON(http.StatusConflict, Rerror("Token not valid"))
	}
}

func workloads(c *gin.Context){
	params := strings.Split(c.Request.Header.Get("Authorization"), " ")
	token := params[1]

	if _, ok := Users[token]; ok {
		workloadName := c.PostForm("workload_name")
		filter := c.PostForm("filter")

		if strings.Contains(workloadName, "_"){
			c.JSON(http.StatusConflict, Rerror("invalid char _"))
		}
		if strings.Contains(workloadName, "="){
			c.JSON(http.StatusConflict, Rerror("Invalid char ="))
		}

		taken := false
		for _, v := range controller.Workloads {
			if v.Name == workloadName {
				taken = true
				break
			}
		}
		if (!taken){
			workloadStatus := "scheduling"
			if len(controller.Workers) > 0 {
				workloadStatus = "running"
			}
			uploadsFolder := "public/results/" + workloadName + "/"
			_ = os.MkdirAll(uploadsFolder, 0755)

			downloadFolder := "download/" + workloadName + "/"
			_ = os.MkdirAll("public/" + downloadFolder, 0755)

			newWL := controller.Workload{
				Id: fmt.Sprintf("%v", len(controller.Workloads)),
				Filter: filter,
				Name: workloadName,
				Status: workloadStatus,
				Jobs: 0,
				Imgs: []string{},
				Filtered: []string{},
			}
			controller.Workloads[fmt.Sprintf("%v", newWL.Id)] = newWL
			c.JSON(http.StatusOK, map[string]interface{}{
				"workload_id": newWL.Id,
				"filter":   filter,
				"workload_name": workloadName,
				"status": newWL.Status,
				"running_jobs": newWL.Jobs,
				"filtered_images": newWL.Filtered,
			})
		}else {
			c.JSON(http.StatusOK, Rerror("Workload already exists"))
		}
	} else {
		c.JSON(http.StatusOK, Rerror("YToken not valid"))
	}
}

func workloadsGet(c *gin.Context) {
	params := strings.Split(c.Request.Header.Get("Authorization"), " ")
	token := params[1]

	workloadId := c.Param("workload_id")
	if _, ok := Users[token]; ok {
		reqWorkload := controller.Workloads[workloadId]
		reqStatus := "running"
		reqJobs := len(reqWorkload.Imgs) - len(controller.Workloads[workloadId].Filtered)
		if len(reqWorkload.Imgs) == len(reqWorkload.Filtered){
			reqStatus = "completed"
		}
		updatedWL := controller.Workload{
			Id: reqWorkload.Id,
			Filter: reqWorkload.Filter,
			Name: reqWorkload.Name,
			Status: reqStatus, 
			Jobs: reqWorkload.Jobs,
			Imgs: reqWorkload.Imgs,
			Filtered: reqWorkload.Filtered,
		}
		controller.Workloads[workloadId] = updatedWL
		c.JSON(http.StatusOK, map[string]interface{}{
			"workload_id": updatedWL.Id,
			"filter":   updatedWL.Filter,
			"workload_name": updatedWL.Name,
			"status": updatedWL.Status,
			"running_jobs": reqJobs,
			"filtered_images": controller.Workloads[workloadId].Filtered,
		})
	} else {
		c.JSON(http.StatusConflict, Rerror("Token not valid"))
	}
}

func images(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "application/json")
	fmt.Println("Response Type:", c.Writer.Header().Get("Content-Type"))
	params := strings.Split(c.Request.Header.Get("Authorization"), " ")
	token := params[1]
	if _, ok := Users[token]; ok {
		file, err := c.FormFile("data")
		if err != nil {
			c.JSON(http.StatusConflict, Rerror("Error retrieving data"))
			return
		}
		workloadId := c.PostForm("workload_id")
		id := 0
		myWorkload := controller.Workload{}
		updatedWL := controller.Workload{}
		if _, ok := controller.Workloads[workloadId]; ok {
			updatedWL = controller.Workload{
				Id: controller.Workloads[workloadId].Id,
				Filter: controller.Workloads[workloadId].Filter,
				Name: controller.Workloads[workloadId].Name,
				Status: "scheduling",
				Jobs: controller.Workloads[workloadId].Jobs + 1,
				Imgs: controller.Workloads[workloadId].Imgs,
			}
			myWorkload = updatedWL
			id = len(controller.Workloads[workloadId].Imgs) + 1
		} else {
			c.JSON(http.StatusConflict, Rerror("Invalid workload"))
		}
		fileId := fmt.Sprintf("o%v_%v", id, updatedWL.Name) 
		newFilename := fileId + filepath.Ext(file.Filename)
		downloadFolder := "public/download/" + myWorkload.Name + "/"
		newPath := path.Join(downloadFolder, newFilename)
		updatedWL.Imgs = append(controller.Workloads[workloadId].Imgs, newFilename)
		controller.Workloads[workloadId] = updatedWL
		if err := c.SaveUploadedFile(file, newPath); err != nil {
			c.JSON(http.StatusConflict, Rerror("File not saved"))
			return
		}
		registeredImage := controller.Image{Id:id, Name: fileId, Ext: filepath.Ext(file.Filename)}
		controller.Uploads[fileId] = registeredImage
		details := [4]string{newPath, filepath.Ext(file.Filename), workloadId, controller.Workloads[workloadId].Filter}
		sampleJob := scheduler.Job{Address: "localhost:50051", RPCName: "image", Info: details}
		Jobs <- sampleJob
		cantTests += 1
		time.Sleep(time.Second * 1)
		c.JSON(http.StatusOK, Rsubida(workloadId, fileId, "original"))
	} else {
		c.JSON(http.StatusConflict, Rerror("Token not valid"))
	}
}

func ImagesGet(c *gin.Context) {
	params := strings.Split(c.Request.Header.Get("Authorization"), " ")
	token := params[1]
	image_id := c.Param("image_id")
	if _, ok := Users[token]; ok {
		if string(image_id[0]) == "f" {
			imgInfo := strings.Split(image_id, "_")
			downloadPath := "./public/results/" + imgInfo[1] + "/" + image_id + ".png"
			c.File(downloadPath)
		} else {
			ext := controller.Uploads[image_id].Ext
			imgInfo := strings.Split(image_id, "_")
			downloadPath := "./public/download/" + imgInfo[1] + "/" + image_id + ext
			c.File(downloadPath)
		}
	} else {
		c.JSON(http.StatusConflict, Rerror("Token not valid"))
	}
}

func Rsubida(workloadId string, id string, imgType string) (gin.H) {
	resp := gin.H{
		"workload_id": workloadId,
		"image_id": id,
		"type": imgType,
	}
	return resp
}

func Rstatus() (gin.H) {
	t := time.Now()
	resp := gin.H{
		"system_name": "Distributed Parallel Image Processing (DPIP) System",
		"server_time": t.Format("2006-01-02 15:04:05"),
		"active_workloads": len(controller.Workloads),
	}
	return resp
}

func RLogout(user string) (gin.H) {
	msg := fmt.Sprintf("Bye %v, now your token has been revoked", user)
	resp := gin.H{
		"logout_message": msg,
	}
	return resp
}

func Rerror(msg string) (gin.H) {
	resp := gin.H{
		"status": "error",
		"message": msg,
	}
	return resp
}

func Rlogin(user string, token string) (gin.H) {
	resp := gin.H{
		"user": user,
		"token": token,
	}
	return resp
}

func Start() {
	r := gin.Default()
	r.POST("/login", login)
	r.DELETE("/logout", logout)
	r.GET("/status", status)
	r.POST("/images", images)
	r.GET("/images/:image_id", ImagesGet)
	r.POST("/workloads", workloads)
	r.GET("/workloads/:workload_id", workloadsGet)
	r.Run("localhost:8080")
}

