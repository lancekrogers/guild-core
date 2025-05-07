# Task Execution Flow

How tasks move from definition to completion in Guild.

## Phases

1. **Spec → Task**: Parse `.md` spec files into tasks.
2. **Assignment**: Assign to best-fit agent via cost model.
3. **Execution**: Agent performs the work.
4. **Review/Test**: Another agent verifies result.
5. **Merge**: Output committed via Git workflow.
