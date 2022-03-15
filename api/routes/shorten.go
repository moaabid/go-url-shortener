package routes

import (
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/moaabid/go-url-shortener/database"
	"github.com/moaabid/go-url-shortener/helpers"
)

type request struct {
	URL         string        `json:"url"`
	CustomShort string        `json:"custom_short"`
	Expiry      time.Duration `json:"expiry"`
}

type reponse struct {
	URL             string        `json:"url"`
	CustomShort     string        `json:"custom_short"`
	Expiry          time.Duration `json:"expiry"`
	XRateRemaning   int           `json:"x_rate_remaining"`
	XRateLimitReset time.Duration `json:"x_rate_limit_reset"`
}

func ShortenURL(c *fiber.Ctx) error {
	body := new(request)

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	//implement rate limiting
	r2 := database.CreateClient(1)

	defer r2.Close()

	val, err := r2.Get(database.Ctx, c.IP()).Result()

	if err == nil {
		_ = r2.Set(database.Ctx, c.IP(), os.Getenv("API_QUOTA"), 30*60*time.Second).Err()
	} else {
		val, _ = r2.Get(database.Ctx, c.IP()).Result()
		valInt, _ := strconv.Atoi(val)

		if valInt <= 0 {

			limit, _ := r2.TTL(database.Ctx, c.IP()).Result()
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":              "Rate limit exceeded",
				"x_rate_limit_reset": limit / time.Nanosecond / time.Minute,
			})
		}

	}

	//check if the input is valid
	if !govalidator.IsURL(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid URL",
		})
	}

	//check for domain error
	if !helpers.RemoveDomainError(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Domain error",
		})
	}

	//enforce https
	body.URL = helpers.EnforceHTTP(body.URL)

	r2.Decr(database.Ctx, c.IP())
}
