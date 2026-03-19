package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/npc1607/arch-linux-agent/pkg/logger"
)

// WeatherTool 天气查询工具
type WeatherTool struct {
	client    *http.Client
	baseURL   string
	userAgent string
}

// NewWeatherTool 创建天气查询工具
func NewWeatherTool() *WeatherTool {
	return &WeatherTool{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL:   "https://wttr.in",
		userAgent: "curl",
	}
}

// WeatherCondition 天气状况
type WeatherCondition struct {
	Text        string  `json:"text"`
	Code        int     `json:"code"`
	Temperature int     `json:"temperature"` // 摄氏度
	FeelsLike   int     `json:"feels_like"`  // 体感温度
	Humidity    int     `json:"humidity"`    // 湿度百分比
	WindSpeed   float64 `json:"wind_speed"`  // 风速 km/h
	WindDir     string  `json:"wind_dir"`    // 风向
	UV          float64 `json:"uv"`          // 紫外线指数
	Pressure    int     `json:"pressure"`    // 气压 hPa
	Visibility  float64 `json:"visibility"`  // 能见度 km
}

// Weather 天气信息
type Weather struct {
	Location    string            `json:"location"`
	Country     string            `json:"country"`
	Region      string            `json:"region"`
	Latitude    float64           `json:"latitude"`
	Longitude   float64           `json:"longitude"`
	Timezone    string            `json:"timezone"`
	LocalTime   string            `json:"local_time"`
	Current     WeatherCondition  `json:"current"`
	Forecast    []WeatherForecast `json:"forecast,omitempty"`
}

// WeatherForecast 天气预报
type WeatherForecast struct {
	Date        string            `json:"date"`
	MaxTemp     int               `json:"max_temp"`
	MinTemp     int               `json:"min_temp"`
	Condition   WeatherCondition  `json:"condition"`
	Precipitation float64         `json:"precipitation"` // 降水概率 %
}

// GetWeather 获取指定地点的天气信息
func (w *WeatherTool) GetWeather(ctx context.Context, location string) (*Weather, error) {
	logger.Info("查询天气", logger.String("location", location))

	// 如果没有指定位置，使用基于IP的位置
	if location == "" {
		location = "?format=j1"
	} else {
		// URL编码位置
		location = url.PathEscape(location) + "?format=j1"
	}

	url := fmt.Sprintf("%s/%s", w.baseURL, location)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", w.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("天气服务返回错误状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	weather, err := w.parseWeatherResponse(body)
	if err != nil {
		return nil, fmt.Errorf("解析天气数据失败: %w", err)
	}

	logger.Info("天气查询成功",
		logger.String("location", weather.Location),
		logger.Int("temp", weather.Current.Temperature),
	)

	return weather, nil
}

// parseWeatherResponse 解析wttr.in的JSON响应
func (w *WeatherTool) parseWeatherResponse(body []byte) (*Weather, error) {
	var wttrResp struct {
		Data struct {
			Request []struct {
				Type  string `json:"type"`
				Query string `json:"query"`
			} `json:"request"`
			CurrentCondition []struct {
				TempC            string  `json:"temp_C"`
				FeelsLikeC       string  `json:"FeelsLikeC"`
				Humidity         string  `json:"humidity"`
				WindspeedKmph    string  `json:"windspeedKmph"`
				Winddir16Point   string  `json:"winddir16Point"`
				UVIndex          string  `json:"uvIndex"`
				Pressure         string  `json:"pressure"`
				Visibility       string  `json:"visibility"`
				WeatherDesc      []struct{ Value string } `json:"weatherDesc"`
			} `json:"current_condition"`
			Weather []struct {
				Date   string `json:"date"`
				MaxTempC string `json:"maxtempC"`
				MinTempC string `json:"mintempC"`
				UVIndex string  `json:"uvIndex"`
				Hourly []struct {
					TempC            string  `json:"tempC"`
					FeelsLikeC       string  `json:"FeelsLikeC"`
					Humidity         string  `json:"humidity"`
					WindspeedKmph    string  `json:"windspeedKmph"`
					Winddir16Point   string  `json:"winddir16Point"`
					Pressure         string  `json:"pressure"`
					Visibility       string  `json:"visibility"`
					WeatherDesc      []struct{ Value string } `json:"weatherDesc"`
					ChanceOfRain     string  `json:"chanceofrain"`
				} `json:"hourly"`
			} `json:"weather"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &wttrResp); err != nil {
		return nil, err
	}

	if len(wttrResp.Data.CurrentCondition) == 0 {
		return nil, fmt.Errorf("无效的天气数据格式")
	}

	current := wttrResp.Data.CurrentCondition[0]

	weather := &Weather{
		Location:  "未知位置",  // wttr.in的j1格式不提供位置信息
		Country:   "",
		Region:    "",
		Timezone:  "Asia/Shanghai",
		Current: WeatherCondition{
			Temperature: parseIntSafe(current.TempC),
			FeelsLike:   parseIntSafe(current.FeelsLikeC),
			Humidity:    parseIntSafe(current.Humidity),
			WindSpeed:   parseFloatSafe(current.WindspeedKmph),
			WindDir:     current.Winddir16Point,
			UV:          parseFloatSafe(current.UVIndex),
			Pressure:    parseIntSafe(current.Pressure),
			Visibility:  parseFloatSafe(current.Visibility),
		},
	}

	// 尝试使用英文描述
	if len(current.WeatherDesc) > 0 {
		weather.Current.Text = current.WeatherDesc[0].Value
	}

	// 解析预报（取未来3天）
	if len(wttrResp.Data.Weather) > 0 {
		for i, wday := range wttrResp.Data.Weather {
			if i >= 3 {
				break
			}

			maxTemp := parseIntSafe(wday.MaxTempC)
			minTemp := parseIntSafe(wday.MinTempC)

			// 取中午时段的数据作为预报代表
			var hourData struct {
				TempC          string `json:"tempC"`
				FeelsLikeC     string `json:"FeelsLikeC"`
				Humidity       string `json:"humidity"`
				WindspeedKmph  string `json:"windspeedKmph"`
				Winddir16Point string `json:"winddir16Point"`
				Pressure       string `json:"pressure"`
				Visibility     string `json:"visibility"`
				WeatherDesc    []struct{ Value string } `json:"weatherDesc"`
				ChanceOfRain   string `json:"chanceofrain"`
			}

			if len(wday.Hourly) > 12 {
				hourData = wday.Hourly[12] // 中午12点
			} else if len(wday.Hourly) > 0 {
				hourData = wday.Hourly[0]
			}

			condition := WeatherCondition{
				Temperature: maxTemp,
				FeelsLike:   parseIntSafe(hourData.FeelsLikeC),
				Humidity:    parseIntSafe(hourData.Humidity),
				WindSpeed:   parseFloatSafe(hourData.WindspeedKmph),
				WindDir:     hourData.Winddir16Point,
				UV:          parseFloatSafe(wday.UVIndex),
				Pressure:    parseIntSafe(hourData.Pressure),
				Visibility:  parseFloatSafe(hourData.Visibility),
			}

			if len(hourData.WeatherDesc) > 0 {
				condition.Text = hourData.WeatherDesc[0].Value
			}

			weather.Forecast = append(weather.Forecast, WeatherForecast{
				Date:         wday.Date,
				MaxTemp:      maxTemp,
				MinTemp:      minTemp,
				Condition:    condition,
				Precipitation: parseFloatSafe(hourData.ChanceOfRain),
			})
		}
	}

	return weather, nil
}

// FormatWeather 格式化天气信息为易读文本
func (w *WeatherTool) FormatWeather(weather *Weather) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📍 %s", weather.Location))
	if weather.Region != "" {
		sb.WriteString(fmt.Sprintf(", %s", weather.Region))
	}
	sb.WriteString(fmt.Sprintf(", %s\n\n", weather.Country))

	sb.WriteString("🌤️ 当前天气\n")
	sb.WriteString(fmt.Sprintf("   温度: %d°C (体感 %d°C)\n", weather.Current.Temperature, weather.Current.FeelsLike))
	sb.WriteString(fmt.Sprintf("   天气: %s\n", weather.Current.Text))
	sb.WriteString(fmt.Sprintf("   湿度: %d%%\n", weather.Current.Humidity))
	sb.WriteString(fmt.Sprintf("   风速: %.1f km/h %s\n", weather.Current.WindSpeed, weather.Current.WindDir))
	sb.WriteString(fmt.Sprintf("   气压: %d hPa\n", weather.Current.Pressure))
	sb.WriteString(fmt.Sprintf("   能见度: %.1f km\n", weather.Current.Visibility))
	sb.WriteString(fmt.Sprintf("   紫外线指数: %.1f\n", weather.Current.UV))

	if len(weather.Forecast) > 0 {
		sb.WriteString("\n📅 未来预报\n")
		for _, forecast := range weather.Forecast {
			sb.WriteString(fmt.Sprintf("\n📆 %s\n", forecast.Date))
			sb.WriteString(fmt.Sprintf("   温度: %d°C ~ %d°C\n", forecast.MinTemp, forecast.MaxTemp))
			sb.WriteString(fmt.Sprintf("   天气: %s\n", forecast.Condition.Text))
			sb.WriteString(fmt.Sprintf("   降水概率: %.0f%%\n", forecast.Precipitation))
		}
	}

	return sb.String()
}

// GetWeatherSimple 获取简化版天气信息
func (w *WeatherTool) GetWeatherSimple(ctx context.Context, location string) (string, error) {
	weather, err := w.GetWeather(ctx, location)
	if err != nil {
		return "", err
	}

	return w.FormatWeather(weather), nil
}

// parseIntSafe 安全地解析整数
func parseIntSafe(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

// parseFloatSafe 安全地解析浮点数
func parseFloatSafe(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
