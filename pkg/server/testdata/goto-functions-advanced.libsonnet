local myfunc(arg1, arg2) = {
  arg1: arg1,
  arg2: arg2,
};

{
  accessThroughFunc: myfunc('test', 'test').arg2,
  funcCreatedObj: myfunc('test', 'test'),
  accesThroughFuncCreatedObj: self.funcCreatedObj.arg2,
}
