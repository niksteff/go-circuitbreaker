package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/sony/gobreaker"
)

type RetryError struct {
	Err error
}

func (e *RetryError) Error() string {
	return fmt.Sprintf("retry error: %s", e.Err.Error())
}

func (e *RetryError) Unwrap() error {
	return e.Err
}

func main() {
	// configure a circuit breaker
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:    "test",
		Timeout: 250 * time.Millisecond,
	})

	// instantiate a client
	client := &Client{}

	for i := 0; i < 50; i++ {
		j := i
		func() {
			res, err := Break[Response](cb, func() (Response, error) { return client.Get(j) })
			if err != nil {
				if errors.Is(err, gobreaker.ErrOpenState) {
					err = &RetryError{Err: err}
				}

				if errors.Is(err, gobreaker.ErrTooManyRequests) {
					err = &RetryError{Err: err}
				}

				log.Printf("error response %d: %s", j, err.Error())
				return
			}

			log.Printf("result %d: %v", j, res)
		}()
		time.Sleep(100 * time.Millisecond)
	}
}

type Response string

type Client struct{}

func (c *Client) Get(i int) (Response, error) {
	var err error
	if i < 20 {
		err = fmt.Errorf("error test")
	}
	if err != nil {
		return "", fmt.Errorf("error getting response: %w", err)
	}

	return Response(fmt.Sprintf("%d", i)), nil
}

// Break is a generic function that reliefes the user from performing a type
// assertion manually when using the gobreaker Execute func.
func Break[T any](cb *gobreaker.CircuitBreaker, f func() (T, error)) (T, error) {
	res, err := cb.Execute(func() (interface{}, error) {
		res, err := f()
		if err != nil {
			return nil, fmt.Errorf("error executing circuit breaker func: %w", err)
		}

		return res, nil
	})
	if err != nil {
		return *new(T), err
	}

	return res.(T), nil
}
