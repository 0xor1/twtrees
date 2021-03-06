package main

import (
	"flag"
	"fmt"
	j "github.com/0xor1/json"
	"github.com/0xor1/trees/server/api"
	"github.com/0xor1/trees/server/util/id"
	"io"
	"net/http"
	"os"
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
	var treesHost string
	fs.StringVar(&treesHost, "th", "https://dev.project-trees.com", "the url host of the project trees env")
	var treesUser string
	fs.StringVar(&treesUser, "tu", "daniel.robinson.spam@gmail.com", "user email for project trees env")
	var treesPwd string
	fs.StringVar(&treesPwd, "tp", "My-T35t-Pwd", "user pwd for project trees env")
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
	totalTasksToCreate := (pow(treeK, treeH+1) - 1) / (treeK - 1)
	fmt.Println("N =", totalTasksToCreate)
	fmt.Println("pn =", projectName)

	runTW(inst, user, pwd, projectName, treeK, treeH)
	runTrees(treesHost, treesUser, treesPwd, projectName, treeK, treeH)
}

func runTW(inst, user, pwd, projectName string, treeK, treeH uint) {
	rm := &twReqMaker{
		inst: inst,
		user: user,
		pwd:  pwd,
	}

	fmt.Println("starting in TW")
	fmt.Println("get my id")
	myId := rm.get("/me.json", nil).MustInt64("person", "id")
	fmt.Println("myId =", myId)

	fmt.Println("create project")
	pj := j.MustFromString(`{"project":{}}`)
	pj.MustGet("project").MustSet("name", projectName).MustSet("people", int64Str(myId)).MustSet("use-tasks", true)
	projectId := rm.post("/projects.json", pj).MustInt64("id")
	fmt.Println("projectId =", projectId)

	fmt.Println("create tasklist")
	tasklistId := rm.post(fmt.Sprintf("/projects/%d/tasklists.json", projectId), j.MustNew().MustSet("todo-list", "name", "twtrees")).MustInt64("TASKLISTID")
	fmt.Println("tasklistId =", tasklistId)

	fmt.Println("create root task")
	pj = j.MustFromString(`{"todo-item":{}}`)
	pj.MustGet("todo-item").MustSet("content", "0").MustSet("estimated-minutes", 60).MustSet("start-date", todayDateString()).MustSet("due-date", tomorrowDateString())
	rootTaskId := rm.post(fmt.Sprintf("/tasklists/%d/tasks.json", tasklistId), pj).MustInt64("id")
	fmt.Println("rootTaskId =", rootTaskId)

	start := time.Now()
	twCreatePerfectKaryTree(rm, tasklistId, rootTaskId, 0, 0, treeK, treeH)
	fmt.Println("time to create tree (excluding root node)", time.Now().Sub(start))

	start = time.Now()
	pj = j.MustFromString(`{"todo-item":{}}`)
	pj.MustGet("todo-item").MustSet("start-date", tomorrowDateString()).MustSet("due-date", dayAfterTomorrowDateString()).MustSet("push-subtasks", true).MustSet("push-dependents", true).MustSet("use-defaults", false)
	rm.put(fmt.Sprintf("/tasks/%d.json", rootTaskId), pj)
	fmt.Println("time to push start/due dates", time.Now().Sub(start))
	fmt.Println("finished in TW")
}

func runTrees(host, user, pwd, projectName string, treeK, treeH uint) {
	api, err := api.New(host, user, pwd)
	panicIf(err)
	me := api.Me

	fmt.Println("starting in Trees")

	fmt.Println("create project")
	project , err := api.V1.Project.Create(me.Region, me.Shard, me.Id, projectName, nil, 8, 5, nil, nil, true, false, nil)
	panicIf(err)
	fmt.Println("projectId =", project.Id.String())

	start := time.Now()
	treesCreatePerfectKaryTree(api, project.Id, project.Id, 0, 0, treeK, treeH)
	fmt.Println("time to create tree (excluding root node)", time.Now().Sub(start))
	fmt.Println("finished in Trees")
}

func pow(x, y uint) uint {
	val := x
	for i := uint(0); i < y-1; i++ {
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

func (r *twReqMaker) do(method, path string, body *j.Json) *j.Json {
	var re io.Reader
	if body != nil {
		re = body.MustToReader()
	}
	req, e := http.NewRequest(method, r.inst+path, re)
	panicIf(e)
	req.SetBasicAuth(r.user, r.pwd)
	req.Header.Set("twProjectsVer", "twtrees")
	resp, e := http.DefaultClient.Do(req)
	panicIf(e)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	return j.MustFromReadCloser(resp.Body)
}

func (r *twReqMaker) post(path string, body *j.Json) *j.Json {
	return r.do("POST", path, body)
}

func (r *twReqMaker) put(path string, body *j.Json) *j.Json {
	return r.do("PUT", path, body)
}

func (r *twReqMaker) get(path string, body *j.Json) *j.Json {
	return r.do("GET", path, body)
}

func twCreatePerfectKaryTree(rm *twReqMaker, tasklistId, parentTaskId, lastUsedNameIdx int64, currentDepth, k, h uint) int64 {
	if currentDepth >= h {
		return lastUsedNameIdx
	}
	for i := uint(0); i < k; i++ {
		lastUsedNameIdx++
		fmt.Printf("\rcreating node %d", lastUsedNameIdx)
		pj := j.MustFromString(`{"todo-item":{}}`)
		pj.MustGet("todo-item").MustSet("content", int64Str(lastUsedNameIdx)).MustSet("estimated-minutes", 60).MustSet("start-date", todayDateString()).MustSet("due-date", tomorrowDateString()).MustSet("parentTaskId", parentTaskId)
		taskId := rm.post(fmt.Sprintf("/tasklists/%d/tasks.json", tasklistId), pj).MustInt64("id")
		lastUsedNameIdx = twCreatePerfectKaryTree(rm, tasklistId, taskId, lastUsedNameIdx, currentDepth+1, k, h)
	}
	return lastUsedNameIdx
}

func treesCreatePerfectKaryTree(api *api.API, projectId, parentId id.Id, lastUsedNameIdx int64, currentDepth, k, h uint) int64 {
	if currentDepth >= h {
		return lastUsedNameIdx
	}
	me := api.Me
	var previousSiblingId *id.Id
	for i := uint(0); i < k; i++ {
		lastUsedNameIdx++
		fmt.Printf("\rcreating node %d", lastUsedNameIdx)
		trueVal := true
		falseVal := false
		isAbstract := trueVal
		isParallel := &trueVal
		remainingTimeVal := uint64(lastUsedNameIdx)
		var remainingTime *uint64
		if currentDepth == h - 1 {
			//make concrete tasks at the bottom
			isAbstract = false
			isParallel = nil
			remainingTime = &remainingTimeVal
		} else if i % 2 == 0 {
			isParallel = &falseVal
		}
		task, err := api.V1.Task.Create(me.Region, me.Shard, me.Id, projectId, parentId, previousSiblingId, int64Str(lastUsedNameIdx), nil, isAbstract, isParallel, nil, remainingTime)
		panicIf(err)
		previousSiblingId = &task.Id
		lastUsedNameIdx = treesCreatePerfectKaryTree(api, projectId, task.Id, lastUsedNameIdx, currentDepth+1, k, h)
	}
	return lastUsedNameIdx
}

func int64Str(i int64) string {
	return fmt.Sprintf("%d", i)
}

func panicIf(e error) {
	if e != nil {
		panic(e)
	}
}
