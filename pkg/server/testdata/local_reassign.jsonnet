local job = {
  steps: { name: 'a', value: 'b' },
};

local newjob = job + {
  steps: {},
  step: super.steps,
};

{
  a: newjob,
}
