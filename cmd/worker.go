/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/knightfall22/matrose/worker"
	"github.com/spf13/cobra"
)

// workerCmd represents the worker command
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Worker command to operate a Matrose worker node.",
	Long: `matrose worker command. 

	The worker runs tasks and responds to the manager's requests about task state.`,
	Run: func(cmd *cobra.Command, args []string) {
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")
		name, _ := cmd.Flags().GetString("name")
		dbType, _ := cmd.Flags().GetString("dbtype")

		log.Println(dbType)
		log.Println("Starting worker.")
		w, err := worker.New(name, dbType)
		if err != nil {
			log.Panicf("could not start worker: %v\n", err)
		}

		api := worker.Api{Address: host, Port: port, Worker: w}

		go w.RunTask()
		go w.CollectStats()
		go w.UpdateTasks()
		log.Printf("Starting worker API on http://%s:%d", host, port)
		api.StartServer()
		time.Sleep(5 * time.Second)
	},
}

func init() {
	rootCmd.AddCommand(workerCmd)

	workerCmd.Flags().StringP("host", "H", "0.0.0.0", "Hostname or IP address")

	workerCmd.Flags().IntP("port", "p", 5556, "Hostname or IP address")

	workerCmd.Flags().StringP("name", "n", fmt.Sprintf("worker-%s", uuid.New().String()), "Name of the worker")

	workerCmd.Flags().StringP("dbtype", "d", "memory", "Type of datastore  to use for tasks (\"memory\" or \"persistent\")")

}
