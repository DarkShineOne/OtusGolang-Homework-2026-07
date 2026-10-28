package hw10programoptimization

import (
	"bufio"
	"bytes"
	"io"
	"strings"

	"github.com/buger/jsonparser" //nolint:depguard
)

type DomainStat map[string]int

func GetDomainStat(r io.Reader, domain string) (DomainStat, error) {
	result := make(DomainStat)

	scanner := bufio.NewScanner(r)
	// размер буффера 10мб для больших по размеру данных
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	suffix := "." + strings.ToLower(domain)
	targetDomain := strings.ToLower(domain)

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		// github.com/buger/jsonparser убирает аллокации в структуры
		email, dataType, _, err := jsonparser.Get(line, "Email")
		if err != nil || dataType != jsonparser.String {
			continue
		}

		email = bytes.TrimSpace(email)
		if len(email) == 0 {
			continue
		}

		atIdx := bytes.IndexByte(email, '@')
		if atIdx == -1 {
			continue
		}

		domainPart := email[atIdx+1:]
		for i := 0; i < len(domainPart); i++ {
			if domainPart[i] >= 'A' && domainPart[i] <= 'Z' {
				domainPart[i] += 32
			}
		}

		lowerDomain := string(domainPart)
		if strings.HasSuffix(lowerDomain, suffix) || lowerDomain == targetDomain {
			result[lowerDomain]++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
