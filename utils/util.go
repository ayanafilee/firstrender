package utils

import (
	"fmt"
	"os"
)

func PrintEnv() {
	fmt.Println("Utils package -> DB_USER:", os.Getenv("PORT"))
}
