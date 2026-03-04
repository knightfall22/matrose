package cmd

import (
	"log"
	"time"

	"github.com/knightfall22/matrose/manager"
	"github.com/spf13/cobra"
)

// managerCmd represents the manager command
var managerCmd = &cobra.Command{
	Use:   "manager",
	Short: "Manager command to operate a Matrose manager node.",
	Long: `matrose manager command.
The manager controls the orchestration system and is responsible for: 

- Accepting tasks from users 
- Scheduling tasks onto worker nodes 
- Rescheduling tasks in the event of a node failure 
- Periodically polling workers to get task updates`,
	Run: func(cmd *cobra.Command, args []string) {
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")
		workers, _ := cmd.Flags().GetStringSlice("workers")
		scheduler, _ := cmd.Flags().GetString("scheduler")
		dbType, _ := cmd.Flags().GetString("dbtype")

		log.Println(dbType)
		log.Println("Starting manager.")
		m, err := manager.New(workers, scheduler, dbType)
		if err != nil {
			log.Panicf("could not start worker: %v\n", err)
		}
		api := manager.Api{Address: host, Port: port, Manager: m}
		go m.ProcessTasks()
		go m.UpdateTasks()
		go m.DoHealthChecks()
		log.Printf("Starting manager API on http://%s:%d", host, port)
		api.StartServer()
		time.Sleep(5 * time.Second)
	},
}

func init() {
	rootCmd.AddCommand(managerCmd)

	managerCmd.Flags().StringP("host", "H", "0.0.0.0", "Hostname or IP address")

	managerCmd.Flags().IntP("port", "p", 5555, "Hostname or IP address")

	managerCmd.Flags().StringSliceP("workers", "w",
		[]string{"localhost:5556"},
		"List of workers on which the manager will schedule tasks.",
	)

	managerCmd.Flags().StringP("scheduler", "s", "epvm", "Name of scheduler to use.")

	managerCmd.Flags().StringP("dbtype", "d", "memory", "Type of datastore  to use for tasks (\"memory\" or \"persistent\")")

}
