package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"gocv.io/x/gocv"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("How to run:\n\tfacedetect [camera ID] [classifier XML file]")
		return
	}

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
	img := gocv.NewMat()
	defer img.Close()

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
	for {
		if ok := webcam.Read(&img); !ok {
			fmt.Printf("cannot read device %d\n", deviceID)
			return
		}
		if img.Empty() {
			continue
		}

		// detect faces
		rects := classifier.DetectMultiScale(img)
		if len(rects) > 0 {
			fmt.Printf("found %d faces\n", len(rects))
			name, err := detectFace(img)
			if err != nil {
				fmt.Println(err)
			}
			// draw a rectangle around each face on the original image,
			// along with text identifying as "Human"
			for _, r := range rects {
				gocv.Rectangle(&img, r, blue, 3)

				size := gocv.GetTextSize(name, gocv.FontHersheyPlain, 1.2, 2)
				pt := image.Pt(r.Min.X+(r.Min.X/2)-(size.X/2), r.Min.Y-2)
				gocv.PutText(&img, name, pt, gocv.FontHersheyPlain, 1.2, blue, 2)
			}
		}

		// show the image in the window, and wait 1 millisecond
		window.IMShow(img)
		if window.WaitKey(5) >= 0 {
			break
		}
	}
}

func detectFace(img gocv.Mat) (string, error) {
	frame, err := img.ToImage()
	if err != nil {
		return "", err
	}
	buffer := new(bytes.Buffer)
	err = png.Encode(buffer, frame)
	if err != nil {
		return "", err
	}

	ctx := context.Background()

	sess := session.Must(session.NewSession())
	config := &aws.Config{
		Region: aws.String(endpoints.UsWest2RegionID),
	}
	bytes, err := ioutil.ReadAll(buffer)
	if err != nil {
		return "", err
	}
	rekognitionService := rekognition.New(sess, config)
	image := rekognition.Image{Bytes: []byte(bytes)}
	threshold := 0.7
	output, err := rekognitionService.SearchFacesByImageWithContext(ctx, &rekognition.SearchFacesByImageInput{
		Image:              &image,
		CollectionId:       aws.String("gopaloalto"),
		FaceMatchThreshold: &threshold,
	})
	fmt.Printf("Err: %s\n", err)
	fmt.Printf("Out: %#v\n", output)
	os.Exit(1)
	if err != nil {
		return "", err
	}

	return "Anders Janmyr", nil
}
