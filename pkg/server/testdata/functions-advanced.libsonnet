local myfunc(arg1, arg2) = {
  arg1: arg1,
  arg2: arg2,

  builderPattern(arg3):: self + {
    arg3: arg3,
  },
};

{
  withMixin(arg4):: {
    arg4: arg4,
  },
  funcCreatedObj: myfunc('test1', 'test2').builderPattern('test3') + self.withMixin('test4'),

  accessThroughFunc: myfunc('test1', 'test2').arg1,
  accessThroughFuncCreatedObj: self.funcCreatedObj.arg2,
  accessBuilderPatternThroughFuncCreatedObj: self.funcCreatedObj.arg3,
  accessMixinThroughFuncCreatedObj: self.funcCreatedObj.arg4,

}
