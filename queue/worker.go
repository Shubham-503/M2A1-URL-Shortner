package queue

// import (
// 	"M2A1-URL-Shortner/types"
// 	"M2A1-URL-Shortner/utils"
// 	"fmt"
// )

// // TaskQueue is a buffered in-memory queue of functions to run.
// // var TaskQueue = make(chan types.Task, 100)
// // var LogUploadQueue = make(chan func(), 100)
// // var NotifyAdminQueue = make(chan func(), 100)

// var EventQueue = make(chan types.Task, 100)

// var Subscribers = map[string][]func(interface{}){
// 	"image_uploaded": {
// 		utils.CheckThumbnail,
// 		utils.LogUpload,
// 		utils.NotifyAdmin,
// 	},
// }

// // Worker that picks tasks and calls subscriber functions
// func StartEventWorker() {
// 	go func() {
// 		for t := range EventQueue {
// 			if handlers, ok := Subscribers[t.Event]; ok {
// 				for _, handler := range handlers {
// 					go handler(t.Data) // Call each handler asynchronously
// 				}
// 			} else {
// 				fmt.Printf("No subscriber for event: %s\n", t.Event)
// 			}
// 		}
// 	}()
// }

// StartWorker launches a background goroutine that processes queued tasks.
// func StartWorker(name string) {
// 	go func(name string) {
// 		for task := range TaskQueue {
// 			fmt.Printf("%s executating %+v ", name, task)
// 			switch task.Event {
// 			case "image_uploaded":
// 				// imageData := task.Data.([]byte) // assert type
// 				profileImgBytes, err := os.ReadFile("assets/image/profileImg.jpg")
// 				if err != nil {
// 					fmt.Printf("Error in profilepc read %v", err.Error())
// 				}
// 				utils.CheckThumbnail(profileImgBytes)
// 				// task()
// 				TaskQueue <- types.Task{
// 					Event: "log_upload",
// 				}
// 				TaskQueue <- types.Task{
// 					Event: "notify_admin",
// 				}
// 				// utils.LogUpload(task.Event)
// 				// utils.NotifyAdmin(task.Event)
// 				// LogUploadQueue <- utils.LogUpload
// 				// NotifyAdminQueue <- utils.NotifyAdmin
// 				fmt.Printf("%s task completed\n", name)
// 			case "log_upload":
// 				fmt.Println("log_upload from worker")
// 				utils.LogUpload(task.Event)
// 				fmt.Println("log_upload from worker completed")
// 			case "notify_admin":
// 				fmt.Println("notify_admin from worker")
// 				utils.NotifyAdmin(task.Event)
// 				fmt.Println("notify_admin from worker completed")

// 			default:
// 				fmt.Printf("[%s] Unknown task event: %s\n", name, task.Event)

// 			}
// 		}
// 	}(name)
// }
// func StartLogUploadWorker() {
// 	go func() {
// 		for log := range LogUploadQueue {
// 			log()
// 		}
// 	}()
// }
// func StartNotifyAdminWorker() {
// 	go func() {
// 		for notifyAdmin := range NotifyAdminQueue {
// 			notifyAdmin()
// 		}
// 	}()
// }
