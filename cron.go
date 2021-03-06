package robin

import (
	"fmt"
	"sync"
	"time"
)

type intervalUnit int64

const (
	millisecond intervalUnit = 1
	second                   = 1000 * millisecond
	minute                   = 60 * second
	hour                     = 60 * minute
	day                      = 24 * hour
	week                     = 7 * day
)

/*type afterOrBeforeExecuteTask int

const (
	beforeExecuteTask afterOrBeforeExecuteTask = iota
	afterExecuteTask
)*/

type jobModel int

const (
	jobDelay jobModel = iota
	jobEvery
)

var (
	dc = newDelay()
	ec = newEvery()
)

type cronDelay struct {
	fiber Fiber
}

type cronEvery struct {
	fiber Fiber
}

type Job struct {
	fiber                          Fiber
	identifyId                     string
	task                           Task
	taskDisposer                   Disposable
	weekday                        time.Weekday
	atHour                         int
	atMinute                       int
	atSecond                       int
	interval                       int64
	nextTime                       time.Time
	calculateNextTimeAfterExecuted bool
	lock                           sync.Mutex
	maximumTimes                   int64
	disposed                       bool
	duration                       time.Duration
	jobModel
	intervalUnit
}

// The job executes immediately.
func RightNow() *Job {
	return Delay(0)
}

// The job executes will delay N Milliseconds.
func Delay(delayInMs int64) *Job {
	return dc.Delay(delayInMs)
}

// newDelay Constructors
func newDelay() *cronDelay {
	return new(cronDelay).init()
}

func (c *cronDelay) init() *cronDelay {
	c.fiber = NewGoroutineMulti()
	c.fiber.Start()
	return c
}

func newDelayJob(delayInMs int64) *Job {
	j := NewJob(dc.fiber)
	j.jobModel = jobDelay
	j.interval = delayInMs
	j.maximumTimes = 1
	j.intervalUnit = millisecond
	return j //NewJob(dc.fiber).setInterval(delayInMs).Times(1).setJobModel(jobDelay).Milliseconds()
}

// The job executes will delay N Milliseconds.
func (c *cronDelay) Delay(delayInMs int64) *Job {
	return newDelayJob(delayInMs)
}

// EveryCron Constructors
func newEvery() *cronEvery {
	return new(cronEvery).init()
}

func (c *cronEvery) init() *cronEvery {
	c.fiber = NewGoroutineMulti()
	c.fiber.Start()
	return c
}

// EverySunday The job will execute every Sunday .
func EverySunday() *Job {
	return NewJob(ec.fiber).everyWeek(time.Sunday)
}

// EveryMonday The job will execute every Monday
func EveryMonday() *Job {
	return NewJob(ec.fiber).everyWeek(time.Monday)
}

// EveryTuesday The job will execute every Tuesday
func EveryTuesday() *Job {
	return NewJob(ec.fiber).everyWeek(time.Tuesday)
}

// EveryWednesday The job will execute every Wednesday
func EveryWednesday() *Job {
	return NewJob(ec.fiber).everyWeek(time.Wednesday)
}

// EveryThursday The job will execute every Thursday
func EveryThursday() *Job {
	return NewJob(ec.fiber).everyWeek(time.Thursday)
}

// EveryFriday The job will execute every Friday
func EveryFriday() *Job {
	return NewJob(ec.fiber).everyWeek(time.Friday)
}

// EverySaturday The job will execute every Saturday
func EverySaturday() *Job {
	return NewJob(ec.fiber).everyWeek(time.Saturday)
}

// Everyday The job will execute every day
func Everyday() *Job {
	return ec.Every(1).Days()
}

// Every The job will execute every N everyUnit(ex atHour、atMinute、atSecond、millisecond etc..).
func Every(interval int64) *Job {
	return ec.Every(interval)
}

// Every The job will execute every N everyUnit(ex atHour、atMinute、atSecond、millisecond etc..).
func (c *cronEvery) Every(interval int64) *Job {
	j := NewJob(ec.fiber)
	j.interval = interval
	j.intervalUnit = millisecond
	return j
}

// return Job Constructors
func NewJob(fiber Fiber) *Job {
	j := &Job{}
	j.jobModel = jobEvery
	j.maximumTimes = -1
	j.atHour = -1
	j.atMinute = -1
	j.atSecond = -1
	j.fiber = fiber
	return j
}

// Dispose Job's Dispose
func (j *Job) Dispose() {
	if j.getDisposed() {
		return
	}
	j.setDisposed(true)
	j.taskDisposer.Dispose()
}

// Identify Job's Identify
func (j *Job) Identify() string {
	return fmt.Sprintf("%p-%p", &j, &j.fiber)
}

// everyWeek a time interval of execution
func (j *Job) everyWeek(dayOfWeek time.Weekday) *Job {
	j.intervalUnit = week
	j.weekday = dayOfWeek
	j.interval = 1
	return j
}

// Days a time interval of execution
func (j *Job) Days() *Job {
	j.intervalUnit = day
	return j
}

// Hours a time interval of execution
func (j *Job) Hours() *Job {
	j.intervalUnit = hour
	return j
}

// Minutes a time interval of execution
func (j *Job) Minutes() *Job {
	j.intervalUnit = minute
	return j
}

// Seconds a time interval of execution
func (j *Job) Seconds() *Job {
	j.intervalUnit = second
	return j
}

// Milliseconds a time interval of execution
func (j *Job) Milliseconds() *Job {
	j.intervalUnit = millisecond
	return j
}

// At the time specified at execution time
func (j *Job) At(hh int, mm int, ss int) *Job {
	//j.lock.Lock()
	j.atHour = Abs(hh) % 24
	j.atMinute = Abs(mm) % 60
	j.atSecond = Abs(ss) % 60
	//j.lock.Unlock()
	return j
}

// AfterExecuteTask Start timing after the Task is executed
// just for delay model、every N second and every N millisecond
func (j *Job) AfterExecuteTask() *Job {
	if j.jobModel == jobDelay ||
		j.intervalUnit == second ||
		j.intervalUnit == millisecond {
		j.calculateNextTimeAfterExecuted = true
	}
	return j
}

// BeforeExecuteTask Start timing before the Task is executed
func (j *Job) BeforeExecuteTask() *Job {
	j.calculateNextTimeAfterExecuted = false
	return j
}

// Times set the job maximum number of executed times
func (j *Job) Times(times int64) *Job {
	j.maximumTimes = times
	return j
}

// Do some job needs to execute.
func (j *Job) Do(fun interface{}, params ...interface{}) Disposable {
	var duration time.Duration
	j.task = newTask(fun, params...)
	j.duration = time.Duration(j.interval*int64(j.intervalUnit)) * time.Millisecond
	now := time.Now()
	if j.jobModel == jobDelay || j.intervalUnit == second || j.intervalUnit == millisecond {
		duration += j.duration
		j.nextTime = now /*.Add(j.duration)*/
	} else {
		switch j.checkAtTime(now).intervalUnit {
		case week:
			i := (7 - (int(now.Weekday() - j.weekday))) % 7
			j.nextTime = time.Date(now.Year(), now.Month(), now.Day(), j.atHour, j.atMinute, j.atSecond, now.Nanosecond(), time.Local).AddDate(0, 0, int(i))
		case day:
			j.nextTime = time.Date(now.Year(), now.Month(), now.Day(), j.atHour, j.atMinute, j.atSecond, now.Nanosecond(), time.Local)
		case hour:
			j.nextTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), j.atMinute, j.atSecond, now.Nanosecond(), time.Local)
		case minute:
			j.nextTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), j.atSecond, now.Nanosecond(), time.Local)
		}

		if j.interval > 1 {
			duration += j.duration
		}

		if j.nextTime.Before(now) {
			duration += j.duration
		}
	}

	j.nextTime = j.nextTime.Add(duration)
	firstInMs := j.nextTime.Sub(now).Nanoseconds() / time.Millisecond.Nanoseconds()
	j.schedule(firstInMs)
	return j
}

// canDo the job can be execute or not
func (j *Job) canDo() {
	adjustTime := j.nextTime.Sub(time.Now()).Nanoseconds() / time.Millisecond.Nanoseconds()
	if adjustTime > 0 {
		j.schedule(adjustTime)
		return
	}

	if j.calculateNextTimeAfterExecuted {
		s := time.Now()
		j.task.run()
		d := time.Now().Sub(s)
		j.nextTime = j.nextTime.Add(d)
	} else {
		j.fiber.EnqueueWithTask(j.task)
	}

	j.maximumTimes += -1
	if j.maximumTimes == 0 {
		j.Dispose()
		return
	}

	j.nextTime = j.nextTime.Add(j.duration)
	adjustTime = j.nextTime.Sub(time.Now()).Nanoseconds() / time.Millisecond.Nanoseconds()
	j.schedule(adjustTime)
}

func (j *Job) schedule(firstInMs int64) {
	j.lock.Lock()
	j.taskDisposer = j.fiber.Schedule(firstInMs, j.canDo)
	j.lock.Unlock()
}

func (j *Job) getDisposed() bool {
	j.lock.Lock()
	defer j.lock.Unlock()
	return j.disposed
}

func (j *Job) setDisposed(r bool) {
	j.lock.Lock()
	j.disposed = r
	j.lock.Unlock()
}

func (j *Job) checkAtTime(now time.Time) *Job {
	if j.atHour < 0 {
		j.atHour = now.Hour()
	}

	if j.atMinute < 0 {
		j.atMinute = now.Minute()
	}

	if j.atSecond < 0 {
		j.atSecond = now.Second()
	}
	return j
}
