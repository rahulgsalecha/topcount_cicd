package main

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].priority > pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// update modifies the priority and value of an Item in the queue.
func (pq *PriorityQueue) update(item *Item, value string, priority int) {
	item.value = value
	item.priority = priority
	heap.Fix(pq, item.index)
}

type Artifacts struct {
	Artifacts []Artifact `json:"results"`
}

type Artifact struct {
	Repo       string    `json:"repo"`
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Size       int64     `json:"size"`
	Created    time.Time `json:"created"`
	CreatedBy  string    `json:"created_by"`
	Modified   time.Time `json:"modified"`
	ModifiedBy string    `json:"modified_by"`
	Updated    time.Time `json:"updated"`
}

type ArtifactStats struct {
	URI                  string `json:"uri"`
	DownloadCount        int    `json:"downloadCount"`
	LastDownloaded       int64  `json:"lastDownloaded"`
	LastDownloadedBy     string `json:"lastDownloadedBy"`
	RemoteDownloadCount  int32  `json:"remoteDownloadCount"`
	RemoteLastDownloaded int64  `json:"remoteLastDownloaded"`
}

// An Item is something we manage in a priority queue.
type Item struct {
	value    string // The value of the item; arbitrary.
	priority int    // The priority of the item in the queue.
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int // The index of the item in the heap.
}

func getRequest(repo string, path string, name string) int {
	url := "http://104.154.94.138/artifactory/api/storage/" + repo + "/" + path + "/" + name + "?stats="
	//fmt.Println("url:" + url)

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("authorization", "Basic YWRtaW46NDlyTVU4VmpEdA==")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("postman-token", "64efe897-30bb-44a7-9ecf-5a5e336405cc")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	//fmt.Println(res)
	//fmt.Println(string(body))

	var artifactStats ArtifactStats
	json.Unmarshal(body, &artifactStats)
	//fmt.Printf("artifactStats : %v\n", artifactStats)

	return artifactStats.DownloadCount
}

func main() {

	url := "http://104.154.94.138/artifactory/api/search/aql"

	payload := strings.NewReader("items.find(\n{\n        \"repo\":{\"$eq\":\"jcenter-cache\"}\n}\n)")

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("authorization", "Basic YWRtaW46NDlyTVU4VmpEdA==")
	req.Header.Add("cache-control", "no-cache")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	//fmt.Println(res)
	//fmt.Println(string(body))

	var artifacts Artifacts
	json.Unmarshal(body, &artifacts)

	countMap := make(map[string]int)
	for i := 0; i < len(artifacts.Artifacts); i++ {
		if strings.Contains(artifacts.Artifacts[i].Name, ".jar") {
			//fmt.Println("Repo: " + artifacts.Artifacts[i].Repo)
			//fmt.Println("Path: " + artifacts.Artifacts[i].Path)
			//fmt.Println("Name: " + artifacts.Artifacts[i].Name)
			downloadsCount := getRequest(artifacts.Artifacts[i].Repo, artifacts.Artifacts[i].Path, artifacts.Artifacts[i].Name)
			//fmt.Println("downloadsCount:" + fmt.Sprint(downloadsCount))
			//fmt.Println("=====")
			countMap[artifacts.Artifacts[i].Name] = downloadsCount
		}
	}

	pq := make(PriorityQueue, len(countMap))
	i := 0
	for value, priority := range countMap {
		pq[i] = &Item{
			value:    value,
			priority: priority,
			index:    i,
		}
		i++
	}
	heap.Init(&pq)

	// Take the items out; they arrive in decreasing priority order.
	count := 0
	for pq.Len() > 0 && count < 2 {
		item := heap.Pop(&pq).(*Item)
		fmt.Printf("%.2d:%s \n", item.priority, item.value)
		count++
	}

}
