package xgboost

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/z4rd0s/go-xgboost/core"
)

type tester interface {
	Helper()
	Error(args ...interface{})
}

func trainAndSaveModel(t tester) (string, func()) {
	if t != nil {
		t.Helper()
	}
	cols := 3
	rows := 5
	trainData := make([]float32, cols*rows)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			trainData[(i*cols)+j] = float32((i + 1) * (j + 1))
		}
	}

	trainLabels := make([]float32, rows)
	for i := 0; i < rows; i++ {
		trainLabels[i] = float32(1 + i*i*i)
	}

	matrix, err := core.XGDMatrixCreateFromMat(trainData, rows, cols, -1)
	if err != nil && t != nil {
		t.Error(err)
	}

	err = matrix.SetFloatInfo("label", trainLabels)
	if err != nil && t != nil {
		t.Error(err)
	}

	booster, err := core.XGBoosterCreate([]*core.XGDMatrix{matrix})
	if err != nil && t != nil {
		t.Error(err)
	}

	noErr := func(err error) {
		if err != nil && t != nil {
			t.Error(err)
		}
	}

	noErr(booster.SetParam("booster", "gbtree"))
	noErr(booster.SetParam("objective", "reg:linear"))
	noErr(booster.SetParam("max_depth", "5"))
	noErr(booster.SetParam("eta", "0.1"))
	noErr(booster.SetParam("min_child_weight", "1"))
	noErr(booster.SetParam("subsample", "0.5"))
	noErr(booster.SetParam("colsample_bytree", "1"))
	noErr(booster.SetParam("num_parallel_tree", "1"))
	noErr(booster.SetParam("silent", "1"))

	// perform 200 learning iterations
	for iter := 0; iter < 200; iter++ {
		noErr(booster.UpdateOneIter(iter, matrix))
	}

	dir, err := ioutil.TempDir("", "go-xgboost")
	if err != nil {
		log.Fatal(err)
	}

	savePath := path.Join(dir, "testmodel.bst")

	noErr(booster.SaveModel(savePath))

	return savePath, func() {
		os.RemoveAll(dir)
	}
}

func TestBooster(t *testing.T) {
	modelPath, cleanUp := trainAndSaveModel(t)
	defer cleanUp()

	predictor, err := NewPredictor(modelPath, 1, 0, 0, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer predictor.Close(context.TODO())

	cols := 3
	rows := 5
	testData := make([][]float32, rows)
	for i := 0; i < rows; i++ {
		testData[i] = make([]float32, cols)
		for j := 0; j < cols; j++ {
			testData[i][j] = float32((i + 1) * (j + 1))
		}
	}

	expectedResult := []float32{1.0631807, 2.4375393, 8.3054695, 30.843433, 63.097855}

	for i, test := range testData {
		res, err := predictor.Predict(FloatSliceVector(test))
		if err != nil {
			t.Error(err)
		}

		if res[0] != expectedResult[i] {
			t.Error("unexpected result received", expectedResult[i])
		}
	}
}

func BenchmarkBooster(b *testing.B) {
	modelPath, cleanUp := trainAndSaveModel(b)
	defer cleanUp()

	predictor, err := NewPredictor(modelPath, 1, 0, 0, -1)
	if err != nil {
		b.Fatal(err)
	}
	defer predictor.Close(context.TODO())

	testData := []float32{1, 2, 3}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		res, err := predictor.Predict(FloatSliceVector(testData))
		if err != nil {
			b.Error(err)
		}
		if len(res) != 1 {
			b.Error("invalid amount of results received")
		}
	}
}

func BenchmarkBoosterParallel(b *testing.B) {
	modelPath, cleanUp := trainAndSaveModel(b)
	defer cleanUp()

	predictor, err := NewPredictor(modelPath, runtime.NumCPU(), 0, 0, -1)
	if err != nil {
		b.Fatal(err)
	}
	defer predictor.Close(context.TODO())

	testData := []float32{1, 2, 3}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			res, err := predictor.Predict(FloatSliceVector(testData))
			if err != nil {
				b.Error(err)
			}
			if len(res) != 1 {
				b.Error("invalid amount of results received")
			}
		}
	})
}

func ExampleBooster() {
	// Retrieve filepath for a pre-trained model
	modelPath, cleanUp := trainAndSaveModel(nil)
	defer cleanUp()

	// Create predictor and define the number of workers (and other settings)
	predictor, _ := NewPredictor(modelPath, runtime.NumCPU(), 0, 0, -1)
	defer predictor.Close(context.TODO())

	res, _ := predictor.Predict(FloatSliceVector([]float32{1, 2, 3}))
	fmt.Printf("Results: %+v\n", res)
	// output: Results: [1.0631807]
}
