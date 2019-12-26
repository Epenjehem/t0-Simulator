# t0 Simulator

Time0ut Simulator is a package that help you simulate timeout budget.

## How to use

``` Go
simulator := t0simulator.NewSimulator("Subscribe", 600)
simulator.RegisterFunctions(
    t0simulator.NewFunction("Input validation").WithTimeout(20),
    t0simulator.NewFunction("Save to DB").WithDynamicContext(0.5, true),
    t0simulator.NewFunction("Send email").WithDynamicContext(0.5, true),
)
simulator.Run()
```
