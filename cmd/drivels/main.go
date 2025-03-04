package main

import (
	"fmt"
	"log"

	"github.com/trichner/tb/pkg/gdrive"
)

func main() {
	service, err := gdrive.NewDriveService()
	if err != nil {
		log.Fatalf("cannot create service: %v", err)
	}

	drives, err := service.ListDrives()
	if err != nil {
		log.Fatalf("cannot list drives: %v", err)
	}

	fmt.Printf("Kind             Id     Name\n")
	for _, d := range drives {
		fmt.Printf("%3s     %12s \t %s\n", d.Kind, d.Id, d.Name)
	}
}
