package cmd

import (
	"github.com/MShoaei/NewsMiner/database"
	"github.com/spf13/cobra"
)

func newDatabaseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "init-db",
		RunE: func(cmd *cobra.Command, args []string) error {

			dbName, _ := cmd.Flags().GetString("db")
			connString, _ := cmd.Flags().GetString("conn")
			db, err := database.NewDB(dbName, connString)
			if err != nil {
				return err
			}

			db.InitDB()

			return nil
		},
	}

	return cmd
}

func init() {
	rootCmd.AddCommand(newDatabaseCommand())
}
