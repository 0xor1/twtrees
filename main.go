package main

import (
	"flag"
	"os"
	"fmt"
	"net/http"
	j "github.com/0xor1/json"
	"io"
	"time"
)

//util program to generate and manipulate perfect k-ary trees of subtasks in tw projects
func main() {
	// cmdFlags
	fs := flag.NewFlagSet("twtrees", flag.ExitOnError)
	var outputCsvFile bool
	fs.BoolVar(&outputCsvFile, "o", false, "print out a csv file of results")
	var inst string
	fs.StringVar(&inst, "i", "http://sunbeam.teamwork.localhost", "installation url base for running tests on")
	var user string
	fs.StringVar(&user, "u", "donalin+dev1@gmail.com", "user email for basic auth")
	var pwd string
	fs.StringVar(&pwd, "p", "test", "user pwd for basic auth")
	var treeK uint
	fs.UintVar(&treeK, "k", 3, "k-ary tree k value must be >1")
	var treeH uint
	fs.UintVar(&treeH, "h", 3, "k-ary tree h value")
	var projectName string
	fs.StringVar(&projectName, "pn", "twtrees", "project name to use in tw projects")
	fs.Parse(os.Args[1:])
	if treeK < 2 {
		panic("treeK value less than 2")
	}
	projectName += "_" + time.Now().Format("20060102150405")
	fmt.Println("outputCsvFile =", outputCsvFile)
	fmt.Println("i =", inst)
	fmt.Println("u =", user)
	fmt.Println("p =", pwd)
	fmt.Println("k =", treeK)
	fmt.Println("h =", treeH)
	totalTasksToCreate := (pow(treeK, treeH+ 1)-1)/(treeK- 1)
	fmt.Println("N =", totalTasksToCreate)
	fmt.Println("pn =", projectName)
	rm := &twReqMaker{
		inst:      inst,
		user:      user,
		pwd:       pwd,
	}

	fmt.Println("get my id")
	myId := rm.get("/me.json", nil).Int64("person", "id")
	fmt.Println("myId =", myId)

	fmt.Println("create project")
	pj := j.PFromString(`{"project":{}}`)
	pj.Get("project").Set(projectName, "name").Set(int64Str(myId), "people").Set(true, "use-tasks")
	projectId := rm.post("/projects.json", pj).Int64("id")
	fmt.Println("projectId =", projectId)

	fmt.Println("create tasklist")
	tasklistId := rm.post(fmt.Sprintf("/projects/%d/tasklists.json", projectId), j.PNew().Set("twtrees", "todo-list", "name")).Int64("TASKLISTID")
	fmt.Println("tasklistId =", tasklistId)

	fmt.Println("create root task")
	pj = j.PFromString(`{"todo-item":{}}`)
	pj.Get("todo-item").Set("0", "content").Set(60, "estimated-minutes").Set(todayDateString(), "start-date").Set(tomorrowDateString(), "due-date")
	rootTaskId := rm.post(fmt.Sprintf("/tasklists/%d/tasks.json", tasklistId), pj).Int64("id")
	fmt.Println("rootTaskId =", rootTaskId)

	start := time.Now()
	createdTasksInCreationOrder := make([]int64, 0, totalTasksToCreate)
	_, createdTasksInCreationOrder = createPerfectKaryTree(rm, tasklistId, rootTaskId, 0, 0, treeK, treeH, createdTasksInCreationOrder)
	fmt.Println("time to create tree (excluding root node)", time.Now().Sub(start))

	start = time.Now()
	pj = j.PFromString(`{"todo-item":{}}`)
	pj.Get("todo-item").Set(tomorrowDateString(), "start-date").Set(dayAfterTomorrowDateString(), "due-date").Set(true, "push-subtasks").Set(true, "push-dependents").Set(false, "use-defaults")
	rm.put(fmt.Sprintf("/tasks/%d.json", rootTaskId), pj)
	fmt.Println("time to push start/due dates", time.Now().Sub(start))
}

func pow(x, y uint) uint {
	val := x
	for i := uint(0); i < y - 1; i++ {
		x *= val
	}
	return x
}

func todayDateString() string {
	return time.Now().Format("20060102")
}

func tomorrowDateString() string {
	return time.Now().Add(time.Hour * 24).Format("20060102")
}

func dayAfterTomorrowDateString() string {
	return time.Now().Add(time.Hour * 48).Format("20060102")
}

type twReqMaker struct {
	inst      string
	user      string
	pwd       string
	projectId string
}

func (r *twReqMaker) do(method, path string, body *j.PJson) *j.PJson {
	var re io.Reader
	if body != nil {
		re = body.ToReader()
	}
	req, e := http.NewRequest(method, r.inst+path, re)
	panicIf(e)
	req.SetBasicAuth(r.user, r.pwd)
	resp, e := http.DefaultClient.Do(req)
	panicIf(e)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	return j.PFromReadCloser(resp.Body)
}

func (r *twReqMaker) post(path string, body *j.PJson) *j.PJson {
	return r.do("POST", path, body)
}

func (r *twReqMaker) put(path string, body *j.PJson) *j.PJson {
	return r.do("PUT", path, body)
}

func (r *twReqMaker) get(path string, body *j.PJson) *j.PJson {
	return r.do("GET", path, body)
}

func createPerfectKaryTree(rm *twReqMaker, tasklistId, parentTaskId, lastUsedNameIdx int64, currentDepth, k, h uint, createdTaskIdsInCreationOrder[]int64) (int64, []int64) {
	if currentDepth >= h {
		return lastUsedNameIdx, createdTaskIdsInCreationOrder
	}
	for i := uint(0); i < k; i++ {
		lastUsedNameIdx++
		fmt.Println("creating node", lastUsedNameIdx)
		pj := j.PFromString(`{"todo-item":{}}`)
		pj.Get("todo-item").Set(int64Str(lastUsedNameIdx), "content").Set(60, "estimated-minutes").Set(todayDateString(), "start-date").Set(tomorrowDateString(), "due-date").Set(parentTaskId, "parentTaskId")
		taskId := rm.post(fmt.Sprintf("/tasklists/%d/tasks.json", tasklistId), pj).Int64("id")
		createdTaskIdsInCreationOrder = append(createdTaskIdsInCreationOrder, taskId)
		lastUsedNameIdx, createdTaskIdsInCreationOrder = createPerfectKaryTree(rm, tasklistId, taskId, lastUsedNameIdx, currentDepth + 1, k, h, createdTaskIdsInCreationOrder)
	}
	return lastUsedNameIdx, createdTaskIdsInCreationOrder
}

func int64Str(i int64) string {
	return fmt.Sprintf("%d", i)
}

func panicIf(e error) {
	if e != nil {
		panic(e)
	}
}