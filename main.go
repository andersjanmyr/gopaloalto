package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"gocv.io/x/gocv"
)

var name string
var rekognitionService *rekognition.Rekognition

func main() {
	if len(os.Args) < 3 {
		fmt.Println("How to run:\n\tfacedetect [camera ID] [classifier XML file]")
		return
	}

	sess := session.Must(session.NewSession())
	config := &aws.Config{
		Region: aws.String(endpoints.UsWest2RegionID),
	}
	rekognitionService = rekognition.New(sess, config)

	// parse args
	deviceID, _ := strconv.Atoi(os.Args[1])
	xmlFile := os.Args[2]

	// open webcam
	webcam, err := gocv.VideoCaptureDevice(int(deviceID))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer webcam.Close()

	// open display window
	window := gocv.NewWindow("Face Detect")
	defer window.Close()

	// prepare image matrix
	mat := gocv.NewMat()
	defer mat.Close()

	// color for the rect when faces detected
	blue := color.RGBA{0, 0, 255, 0}

	// load classifier to recognize faces
	classifier := gocv.NewCascadeClassifier()
	defer classifier.Close()

	if !classifier.Load(xmlFile) {
		fmt.Printf("Error reading cascade file: %v\n", xmlFile)
		return
	}

	fmt.Printf("start reading camera device: %v\n", deviceID)
	nameChan := make(chan string)
	matChan := make(chan gocv.Mat)
	go startDetectFace(matChan, nameChan)
	go nameLoop(nameChan)
	for {
		if ok := webcam.Read(&mat); !ok {
			fmt.Printf("cannot read device %d\n", deviceID)
			return
		}
		if mat.Empty() {
			continue
		}

		// detect faces
		rects := classifier.DetectMultiScale(mat)
		if len(rects) > 0 {
			fmt.Printf("found %d faces\n", len(rects))
			select {
			case matChan <- mat:
			default:
			}
			for _, r := range rects {
				gocv.Rectangle(&mat, r, blue, 3)
				size := gocv.GetTextSize(name, gocv.FontHersheyPlain, 1.2, 2)
				pt := image.Pt(r.Min.X+(r.Min.X/2)-(size.X/2), r.Min.Y-2)
				gocv.PutText(&mat, name, pt, gocv.FontHersheyPlain, 1.2, blue, 2)
			}
		}

		window.IMShow(mat)
		if window.WaitKey(10) >= 0 {
			break
		}
	}
}

func nameLoop(nameChan <-chan string) {
	for {
		select {
		case name = <-nameChan:
			fmt.Printf("Read name from chan: %s\n", name)
		case <-time.After(9 * time.Second):
			fmt.Println("Timeout name from chan")
			name = "Unknown"
		}
	}
}

func startDetectFace(matrixChan <-chan gocv.Mat, nameChan chan<- string) {
	for {
		mat := <-matrixChan
		fmt.Println("Read mat from chan")
		name, err := detectFace(mat)
		if err != nil {
			fmt.Println(err)
			continue
		}
		if name == "" {
			name = "Unknown"
		}
		nameChan <- name
	}
}

func detectFace(mat gocv.Mat) (string, error) {
	frame, err := mat.ToImage()
	if err != nil {
		return "", err
	}
	buffer := new(bytes.Buffer)
	err = png.Encode(buffer, frame)
	if err != nil {
		return "", err
	}

	ctx := context.Background()

	bytes, err := ioutil.ReadAll(buffer)
	if err != nil {
		return "", err
	}
	image := rekognition.Image{Bytes: []byte(bytes)}
	threshold := 0.7
	start := time.Now()

	output, err := rekognitionService.SearchFacesByImageWithContext(ctx, &rekognition.SearchFacesByImageInput{
		Image:              &image,
		CollectionId:       aws.String("gopaloalto"),
		FaceMatchThreshold: &threshold,
	})
	elapsed := time.Since(start)
	log.Printf("Service call took %s", elapsed)
	if err != nil {
		return "", err
	}

	fmt.Println(output.FaceMatches[0])
	if len(output.FaceMatches) < 1 {
		return "", nil
	}
	return *output.FaceMatches[0].Face.ExternalImageId, nil
}
