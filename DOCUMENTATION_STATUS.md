# Documentation Status and Update Plan

## Current Status (June 4, 2025)

This document tracks the state of Guild's documentation and what needs to be updated post-MVP.

### ✅ Updated Documents
- **README.md** - Updated to reflect current implementation status and available features

### 📝 Documents Requiring Post-MVP Updates

#### High Priority (Update immediately after MVP)
1. **GETTING_STARTED.md**
   - Current: Describes aspirational features and non-existent commands
   - Needed: Working examples, actual command workflows, demo GIFs
   - Waiting for: MVP completion to create accurate demos

2. **docs/getting-started/installation.md**
   - Current: Basic installation steps
   - Needed: Verified installation process, dependency management, troubleshooting

3. **Command Reference** (NEW)
   - Create comprehensive CLI command documentation
   - Document all flags, options, and examples
   - Include real workflow examples

#### Medium Priority (Update within 2 weeks of MVP)
4. **Configuration Guide** (NEW)
   - Document guild.yaml structure
   - Explain all configuration options
   - Provide example configurations

5. **Architecture Documentation** (NEW)
   - Document actual system architecture
   - Component interaction diagrams
   - Data flow explanations

6. **docs/project-structure.md**
   - Verify against actual .guild/ directory structure
   - Add troubleshooting section

#### Lower Priority (Update as features stabilize)
7. **Tool Integration Guide** (NEW)
   - How to add custom tools
   - Tool safety and validation
   - Example tool implementations

8. **Provider Configuration** (NEW)
   - Detailed provider setup guides
   - Cost optimization strategies
   - Multi-provider configurations

9. **Example Projects** (NEW)
   - Create working example projects
   - Include various use cases
   - Step-by-step walkthroughs

### 🚫 Documents to Deprecate/Remove
- References to `guild workshop` command
- Campaign management documentation (until implemented)
- Cost optimization features (until implemented)
- Review workflow documentation (until implemented)

### 📊 Documentation Principles Going Forward

1. **Document What Exists**: Only document implemented features
2. **Clear Roadmap**: Separate planned features into ROADMAP.md
3. **Working Examples**: All code examples must be tested
4. **Version Accuracy**: Keep version numbers and dependencies current
5. **User Journey**: Focus on common workflows and use cases

### 🎯 Post-MVP Documentation Sprint Plan

**Week 1 Post-MVP**:
- Rewrite GETTING_STARTED.md with working examples
- Create command reference
- Update all installation guides
- Add troubleshooting guide

**Week 2 Post-MVP**:
- Create architecture documentation
- Write configuration guide
- Update provider documentation
- Create first example project

**Week 3 Post-MVP**:
- Add tool integration guide
- Create video demos/GIFs
- Write contributor guidelines
- Polish all documentation

### 📝 Notes

- Documentation should be updated only after features are stable
- Focus on accuracy over comprehensiveness initially
- Use real examples from actual Guild usage
- Include common error messages and solutions
- Add medieval theming consistently but don't overdo it