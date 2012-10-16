package main

import (
  "fmt"
  gl "github.com/chsc/gogl/gl21"
  "github.com/runningwild/cmwc"
  "github.com/runningwild/glop/gin"
  "github.com/runningwild/glop/gos"
  "github.com/runningwild/glop/gui"
  "github.com/runningwild/glop/render"
  "github.com/runningwild/glop/system"
  "os"
  "path/filepath"
  "runningwild/pnf"
  "runningwild/tron/base"
  "runtime"
  "runtime/pprof"
)

var (
  sys      system.System
  datadir  string
  ui       *gui.Gui
  wdx, wdy int
  key_map  base.KeyMap
)

func init() {
  runtime.LockOSThread()
  sys = system.Make(gos.GetSystemInterface())

  datadir = filepath.Join(os.Args[0], "..", "..")
  base.SetDatadir(datadir)
  base.Log().Printf("Setting datadir: %s", datadir)
  wdx = 1024
  wdy = 768

  var key_binds base.KeyBinds
  base.LoadJson(filepath.Join(datadir, "key_binds.json"), &key_binds)
  key_map = key_binds.MakeKeyMap()
  base.SetDefaultKeyMap(key_map)
}

func main() {
  sys.Startup()
  err := gl.Init()
  if err != nil {
    panic(err)
  }

  render.Init()
  render.Queue(func() {
    sys.CreateWindow(10, 10, wdx, wdy)
    sys.EnableVSync(true)
    err := gl.Init()
    if err != nil {
      panic(err)
    }
  })
  runtime.GOMAXPROCS(2)
  ui, err = gui.Make(gin.In(), gui.Dims{wdx, wdy}, filepath.Join(datadir, "fonts", "skia.ttf"))
  if err != nil {
    panic(err)
  }

  anchor := gui.MakeAnchorBox(gui.Dims{wdx, wdy})
  ui.AddChild(anchor)
  sys.Think()
  var g Game
  g.Rng = cmwc.MakeCmwc(4224759397, 3)
  g.Rng.SeedWithDevRand()
  g.Dx = 900
  g.Dy = 600
  g.Max_turn = 0.05
  g.Max_acc = 0.25
  g.Friction = 0.95
  var p Player
  p.Alive = true
  p.X = float64(g.Dx) / 2
  p.Y = float64(g.Dy) / 2
  p.Color.R = 255
  p.Max_rate = 10
  p.Influence = 75
  p.Dominance = 10
  g.Players = append(g.Players, p)
  g.GenerateNodes(1000)
  var engine *pnf.Engine
  engine = pnf.NewLocalEngine(&g, 10)
  anchor.AddChild(&GameWindow{Engine: engine}, gui.Anchor{0.5, 0.5, 0.5, 0.5})
  var v float64
  var profile_output *os.File
  var num_mem_profiles int
  ui.AddChild(base.MakeConsole())
  for key_map["quit"].FramePressCount() == 0 {
    sys.Think()
    render.Queue(func() {
      ui.Draw()
    })
    render.Queue(func() {
      sys.SwapBuffers()
    })
    render.Purge()
    up := gin.In().GetKey(gin.Up).FramePressAvg()
    down := gin.In().GetKey(gin.Down).FramePressAvg()
    left := gin.In().GetKey(gin.Left).FramePressAvg()
    right := gin.In().GetKey(gin.Right).FramePressAvg()
    engine.ApplyEvent(Accelerate{0, 2 * (up - down)})
    engine.ApplyEvent(Turn{0, (left - right) / 10})

    if key_map["cpu profile"].FramePressCount() > 0 {
      if profile_output == nil {
        profile_output, err = os.Create(filepath.Join(datadir, "cpu.prof"))
        if err == nil {
          err = pprof.StartCPUProfile(profile_output)
          if err != nil {
            fmt.Printf("Unable to start CPU profile: %v\n", err)
            profile_output.Close()
            profile_output = nil
          }
          fmt.Printf("profout: %v\n", profile_output)
        } else {
          fmt.Printf("Unable to start CPU profile: %v\n", err)
        }
      } else {
        pprof.StopCPUProfile()
        profile_output.Close()
        profile_output = nil
      }
    }

    if key_map["mem profile"].FramePressCount() > 0 {
      f, err := os.Create(filepath.Join(datadir, fmt.Sprintf("mem.%d.prof", num_mem_profiles)))
      if err != nil {
        base.Error().Printf("Unable to write mem profile: %v", err)
      }
      pprof.WriteHeapProfile(f)
      f.Close()
      num_mem_profiles++
    }

    v += 0.01
  }
}
