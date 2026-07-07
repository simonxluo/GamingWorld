package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/simonxluo/GamingWorld/internal/env"
	"github.com/simonxluo/GamingWorld/runtime/llm"
)

func main() {
	if err := env.LoadEnv("/home/user/workspace/GamingWorld/.env"); err != nil {
		fmt.Fprintln(os.Stderr, "warn: 未加载 .env:", err)
	}

	c := &llm.Client{
		BaseUrl: os.Getenv("LLM_BASE_URL"),
		ApiKey:  os.Getenv("LLM_API_KEY"),
		Model:   os.Getenv("LLM_MODEL"),
	}

	fmt.Print("> ")
	sc := bufio.NewScanner(os.Stdin)

	if !sc.Scan() {
		return
	}

	reply, err := c.Complete(context.Background(), "你是一个叫mimi的ai助手", sc.Text())

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Println(reply)

}
