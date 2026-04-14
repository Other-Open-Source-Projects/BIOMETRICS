# BIOMETRICS Coding Standards

**Version:** 1.0  
**Status:** ACTIVE  
**Effective Date:** 2026-02-20  
**Last Updated:** 2026-02-20

---

## Table of Contents

1. [TypeScript Standards](#1-typescript-standards)
2. [Go Standards](#2-go-standards)
3. [Error Handling](#3-error-handling)
4. [Code Style](#4-code-style)
5. [Testing Standards](#5-testing-standards)
6. [Performance Standards](#6-performance-standards)
7. [Security Standards](#7-security-standards)
8. [Documentation Standards](#8-documentation-standards)
9. [Git Workflow](#9-git-workflow)
10. [CI/CD Standards](#10-cicd-standards)

---

## 1. TypeScript Standards

### 1.1 Strict Mode (MANDATORY)

**Rule:** TypeScript strict mode MUST be enabled in all projects.

**Why:** Strict mode prevents common programming errors, enables better type inference, and makes refactoring safer. It catches errors at compile time rather than runtime.

**Configuration (tsconfig.json):**

```json
{
  "compilerOptions": {
    "strict": true,
    "noImplicitAny": true,
    "noImplicitThis": true,
    "alwaysStrict": true,
    "strictNullChecks": true,
    "strictFunctionTypes": true,
    "strictBindCallApply": true,
    "strictPropertyInitialization": true,
    "noImplicitReturns": true,
    "noFallthroughCasesInSwitch": true,
    "noUncheckedIndexedAccess": true,
    "noImplicitOverride": true,
    "noPropertyAccessFromIndexSignature": true
  }
}
```

**DO:**

```typescript
// DONE: CORRECT: Explicit types
function calculateBMI(weight: number, height: number): number {
  if (weight <= 0 || height <= 0) {
    throw new Error('Weight and height must be positive');
  }
  return weight / (height * height);
}

// DONE: CORRECT: Strict null checks
interface User {
  id: string;
  name: string;
  email?: string; // Optional - explicitly marked
}

function getUserEmail(user: User): string {
  return user.email ?? 'No email provided';
}
```

**DON'T:**

```typescript
// ERROR: WRONG: Implicit any
function processData(data) {  // No type for 'data'!
  return data.value * 2;
}

// ERROR: WRONG: Missing null checks
function getUserName(user: User): string {
  return user.email.toUpperCase(); // Could crash if email is undefined!
}

// ERROR: WRONG: Using any
const result: any = fetchData(); // Loses all type safety!
```

### 1.2 No `any` Types

**Rule:** Avoid `any` type unless absolutely necessary with explicit justification.

**Why:** The `any` type defeats the purpose of TypeScript's type system. It makes refactoring dangerous and hides potential bugs. According to 2026 Best Practices, `any` should only be used when dealing with truly dynamic data that cannot be typed.

**Acceptable Use Cases for `any`:**

```typescript
// 1. Third-party libraries without types
// eslint-disable-next-line @typescript-eslint/no-explicit-any
import { SomeLibrary } from 'unknown-library';

// 2. JSON.parse results (runtime dynamic data)
function parseUserData(json: string): Record<string, unknown> {
  const parsed = JSON.parse(json);
  // Explicitly mark as 'any' with comment explaining why
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return parsed as any;
}

// 3. Generic placeholder for future implementation
interface Cache<T = unknown> {
  get(key: string): Promise<T | undefined>;
  set(key: string, value: T): Promise<void>;
}
```

**Better Alternatives to `any`:**

```typescript
// Instead of 'any', use:
interface UnknownData {
  [key: string]: unknown;
}

// OR use 'unknown' (type-safe any)
function processUnknown(value: unknown): string {
  if (typeof value === 'string') {
    return value.toUpperCase(); // TypeScript knows it's a string here
  }
  if (typeof value === 'number') {
    return value.toString();
  }
  throw new Error('Unsupported type');
}

// OR use generics
function identity<T>(value: T): T {
  return value;
}
```

### 1.3 No `@ts-ignore` / `@ts-expect-error`

**Rule:** Never use `@ts-ignore` or `@ts-expect-error` unless absolutely necessary.

**Why:** These directives suppress TypeScript errors without fixing them. They create hidden bugs and make refactoring dangerous. Always fix the underlying type error instead.

**If Absolutely Necessary (Rare Cases):**

```typescript
// Only use when:
 // 1. Type definitions are outdated and cannot be updated
 // 2. Temporary workaround for library bug
 // 3. Known TypeScript limitation with clear workaround

// Use @ts-expect-error over @ts-ignore (more specific)
function legacyFunction(data: string): void {
  // @ts-expect-error - Legacy API requires number, we need string
  legacyLibrary.process(42);
}

// MUST document why
function complexCalculation(): number {
  // @ts-expect-error - Known TypeScript bug with complex generics
  // Bug report: https://github.com/microsoft/TypeScript/issues/XXXXX
  return (Math.random() * 100) as any;
}
```

### 1.4 JSDoc/TSDoc for Public or Non-Obvious APIs

**Rule:** Use JSDoc/TSDoc for public exports, non-obvious contracts, and behavior that benefits from durable explanation. Routine internal helpers and obvious code do not need boilerplate comments.

**Why:** JSDoc/TSDoc works best when it adds durable value: public API clarity, important constraints, invariants, side effects, and usage guidance. Boilerplate comments on obvious code create noise and age badly.

**Function Documentation:**

```typescript
/**
 * Calculates the Body Mass Index (BMI) for a given weight and height.
 * 
 * BMI is calculated as: weight (kg) / height (m)²
 * 
 * @param weight - Weight in kilograms (must be > 0)
 * @param height - Height in meters (must be > 0)
 * @returns The BMI value as a number
 * @throws Error if weight or height is less than or equal to 0
 * 
 * @example
 * ```typescript
 * const bmi = calculateBMI(70, 1.75); // Returns 22.86
 * ```
 */
export function calculateBMI(weight: number, height: number): number {
  if (weight <= 0 || height <= 0) {
    throw new Error('Weight and height must be positive numbers');
  }
  return weight / (height * height);
}
```

**Class Documentation:**

```typescript
/**
 * Manages user authentication and session lifecycle.
 * 
 * This class handles:
 * - User login/logout operations
 * - Session token generation and validation
 * - Password hashing and verification
 * - Refresh token rotation
 * 
 * @remarks
 * This class requires a database connection to be injected.
 * Use {@link AuthManager.withDatabase} for proper initialization.
 * 
 * @example
 * ```typescript
 * const auth = new AuthManager(jwtSecret);
 * await auth.login('user@example.com', 'password123');
 * ```
 */
export class AuthManager {
  private readonly jwtSecret: string;
  private readonly tokenExpiry = 3600; // 1 hour
  
  /**
   * Creates a new AuthManager instance.
   * 
   * @param jwtSecret - Secret key for JWT signing (minimum 32 characters)
   */
  constructor(jwtSecret: string) {
    if (jwtSecret.length < 32) {
      throw new Error('JWT secret must be at least 32 characters');
    }
    this.jwtSecret = jwtSecret;
  }
}
```

**Interface Documentation:**

```typescript
/**
 * Configuration options for the biometric authentication service.
 */
export interface BiometricConfig {
  /** List of allowed biometric types (e.g., ['fingerprint', 'face']) */
  allowedTypes: BiometricType[];
  
  /** Minimum confidence score required for authentication (0-100) */
  minConfidenceScore: number;
  
  /** Whether to require fallback authentication */
  requireFallback: boolean;
  
  /** Session timeout in seconds */
  sessionTimeout: number;
  
  /** Maximum failed attempts before lockout */
  maxFailedAttempts: number;
}
```

### 1.5 Interface vs Type - When to Use Which

**Rule:** Use `interface` for object shapes that may be extended; use `type` for unions, intersections, and primitives.

**Why:** In TypeScript, `interface` and `type` have different capabilities and semantics. Interfaces support declaration merging, while types are more flexible for complex type definitions.

**Use `interface` When:**

```typescript
// 1. Defining object shapes that may be extended
interface User {
  id: string;
  name: string;
  email: string;
}

// Can be extended later
interface User {
  avatar?: string;
}

// 2. Creating class contracts
interface Repository<T> {
  findById(id: string): Promise<T | null>;
  findAll(): Promise<T[]>;
  save(entity: T): Promise<T>;
  delete(id: string): Promise<void>;
}

// 3. Defining API response shapes
interface ApiResponse<T> {
  data: T;
  status: number;
  message: string;
}
```

**Use `type` When:**

```typescript
// 1. Union types
type Status = 'pending' | 'approved' | 'rejected';

type Result<T, E = Error> =
  | { ok: true; value: T }
  | { ok: false; error: E };

// 2. Intersection types
type AdminUser = User & {
  permissions: string[];
  role: 'admin' | 'superadmin';
};

// 3. Tuple types
type Coordinates = [number, number];
type RGB = [number, number, number];

// 4. Primitive aliases
type UserId = string;
type Timestamp = number;

// 5. Utility types
type Readonly<T> = {
  readonly [P in keyof T]: T[P];
};

type Partial<T> = {
  [P in keyof T]?: T[P];
};
```

---

## 2. Go Standards

### 2.1 Error Handling Pattern

**Rule:** Always handle errors explicitly. Never ignore errors. Use the idiomatic Go error handling pattern.

**Why:** Go's error handling is explicit by design. Ignoring errors leads to silent failures that are difficult to debug. Following the idiomatic pattern makes code readable and maintainable.

**DO - Idiomatic Error Handling:**

```go
// DONE: CORRECT: Explicit error handling
func fetchUser(id string) (*User, error) {
    user, err := db.QueryUser(id)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch user %s: %w", id, err)
    }
    return user, nil
}

// DONE: CORRECT: Early return for error cases
func processData(data []byte) (Result, error) {
    if len(data) == 0 {
        return Result{}, errors.New("empty data slice")
    }
    
    parsed, err := parseData(data)
    if err != nil {
        return Result{}, fmt.Errorf("parse error: %w", err)
    }
    
    validated, err := validateData(parsed)
    if err != nil {
        return Result{}, fmt.Errorf("validation error: %w", err)
    }
    
    return transformData(validated)
}

// DONE: CORRECT: Custom error types for better error handling
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
}

// Using custom errors
func validateEmail(email string) error {
    if !isValidEmail(email) {
        return &ValidationError{
            Field:   "email",
            Message: "invalid email format",
        }
    }
    return nil
}
```

**DON'T:**

```go
// ERROR: WRONG: Ignoring errors
func processData(data []byte) {
    parsed, _ := parseData(data)  // Error ignored!
    // What if parseData fails? Silent failure!
}

// ERROR: WRONG: Empty error handling
func fetchUser(id string) *User {
    user, err := db.QueryUser(id)
    if err != nil {
        // Doing nothing with the error!
    }
    return user
}

// ERROR: WRONG: Using panic for normal errors
func processData(data []byte) {
    parsed, err := parseData(data)
    if err != nil {
        panic(err)  // Never do this for recoverable errors!
    }
}
```

### 2.2 Interface Design

**Rule:** Design small, focused interfaces. Prefer composition over large interfaces.

**Why:** Small interfaces are easier to implement, test, and mock. They follow the Go philosophy of composition over inheritance.

**DO:**

```go
// DONE: CORRECT: Small, focused interfaces
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

// Composition - combine small interfaces
type ReadWriter interface {
    Reader
    Writer
}

// DONE: CORRECT: Interface at the point of use
// Define interfaces where they are used, not where they are implemented
func processData(r io.Reader) error {
    data, err := io.ReadAll(r)
    if err != nil {
        return fmt.Errorf("reading data: %w", err)
    }
    // Process data...
    return nil
}

// DONE: CORRECT: Return interfaces, accept concrete types
type UserRepository interface {
    FindByID(id string) (*User, error)
}

func NewUserService(repo UserRepository) *UserService {
    return &UserService{repo: repo}
}

// UserService accepts concrete type, returns interface
func (s *UserService) GetUser(id string) (*User, error) {
    return s.repo.FindByID(id)
}
```

**DON'T:**

```go
// ERROR: WRONG: Large, God interfaces
type EntityManager interface {
    Create(entity interface{}) error
    Read(id string) (interface{}, error)
    Update(entity interface{}) error
    Delete(id string) error
    FindAll() (interface{}, error)
    FindByField(field, value string) (interface{}, error)
    Count() (int, error)
    // ... 50 more methods!
}

// ERROR: WRONG: Defining interfaces in the wrong place
// (package that implements, not package that uses)
package repository

type UserRepository interface {
    FindByID(id string) (*User, error)  // Should be in service package
}
```

### 2.3 Package Structure

**Rule:** Follow standard Go project layout. Group by functionality, not by type.

**Why:** Standard layouts make it easy for Go developers to navigate projects. They also work well with Go tooling.

**Recommended Structure:**

```
myproject/
├── cmd/
│   └── myapp/
│       └── main.go           # Application entry point
├── internal/
│   ├── handler/              # HTTP handlers
│   │   ├── user.go
│   │   └── auth.go
│   ├── service/             # Business logic
│   │   ├── user_service.go
│   │   └── auth_service.go
│   ├── repository/          # Data access
│   │   ├── user_repo.go
│   │   └── db.go
│   └── model/               # Data models
│       └── user.go
├── pkg/                     # Reusable packages (exportable)
│   ├── validator/
│   │   └── validator.go
│   └── logger/
│       └── logger.go
├── api/                    # API definitions (OpenAPI, protobuf)
├── configs/                # Configuration files
├── test/                   # Additional test data
├── go.mod
└── go.sum
```

**Package Naming Rules:**

```go
// DONE: CORRECT: Short, descriptive, lowercase names
package handler    // NOT: httpHandler
package service   // NOT: businessLogicService
package repo      // NOT: repository

// DONE: CORRECT: Use singular form
package user      // NOT: users

// DONE: CORRECT: No underscores in names
package authservice  // NOT: auth_service
```

### 2.4 Testing Requirements

**Rule:** All exported functions MUST have tests. Use table-driven tests for multiple test cases.

**Why:** Tests ensure code works correctly and prevents regressions. Table-driven tests are idiomatic Go and make it easy to add new test cases.

**DO:**

```go
// DONE: CORRECT: Table-driven tests
func TestCalculateBMI(t *testing.T) {
    tests := []struct {
        name     string
        weight   float64
        height   float64
        want     float64
        wantErr  bool
    }{
        {
            name:   "normal case",
            weight: 70,
            height: 1.75,
            want:   22.86,
            wantErr: false,
        },
        {
            name:   "underweight",
            weight: 50,
            height: 1.75,
            want:   16.33,
            wantErr: false,
        },
        {
            name:   "zero weight - error",
            weight: 0,
            height: 1.75,
            want:   0,
            wantErr: true,
        },
        {
            name:   "negative height - error",
            weight: 70,
            height: -1.75,
            want:   0,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := CalculateBMI(tt.weight, tt.height)
            
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            
            assert.NoError(t, err)
            assert.InDelta(t, tt.want, got, 0.01)
        })
    }
}

// DONE: CORRECT: Test packages correctly
// Use the same package name as the tested file
package service

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestUserService_Create(t *testing.T) {
    // Test implementation
}
```

**DON'T:**

```go
// ERROR: WRONG: No tests for exported functions
func CalculateBMI(weight, height float64) float64 {
    return weight / (height * height)
}

// No test file exists!

// ERROR: WRONG: Not using testify/assert
func TestCalculateBMI(t *testing.T) {
    result := CalculateBMI(70, 1.75)
    if result != 22.86 {  // Hard to read error messages
        t.Errorf("expected 22.86, got %f", result)
    }
}

// ERROR: WRONG: Not testing error cases
func TestCalculateBMI(t *testing.T) {
    result := CalculateBMI(70, 1.75)
    if result != 22.86 {
        t.Fail()
    }
    // What about zero weight? Negative height?
}
```

---

## 3. Error Handling

### 3.1 Custom Error Classes

**Rule:** Create custom error types for different error categories. Use error wrapping to preserve context.

**Why:** Custom errors allow callers to handle different error types differently. Error wrapping preserves the error chain for debugging.

**DO:**

```typescript
// DONE: CORRECT: Custom error class hierarchy
export class AppError extends Error {
  constructor(
    message: string,
    public readonly code: string,
    public readonly statusCode: number = 500,
    public readonly cause?: Error
  ) {
    super(message);
    this.name = 'AppError';
    Error.captureStackTrace(this, this.constructor);
  }
}

export class ValidationError extends AppError {
  constructor(message: string, public readonly details?: Record<string, unknown>) {
    super(message, 'VALIDATION_ERROR', 400);
    this.name = 'ValidationError';
  }
}

export class NotFoundError extends AppError {
  constructor(resource: string, id: string) {
    super(`${resource} with id ${id} not found`, 'NOT_FOUND', 404);
    this.name = 'NotFoundError';
  }
}

export class UnauthorizedError extends AppError {
  constructor(message = 'Unauthorized') {
    super(message, 'UNAUTHORIZED', 401);
    this.name = 'UnauthorizedError';
  }
}

// Usage
function fetchUser(id: string): Promise<User> {
  const user = await db.findUser(id);
  if (!user) {
    throw new NotFoundError('User', id);
  }
  return user;
}
```

**DON'T:**

```typescript
// ERROR: WRONG: Using generic errors
function fetchUser(id: string): Promise<User> {
  const user = await db.findUser(id);
  if (!user) {
    throw new Error('User not found'); // No error code, no status code!
  }
  return user;
}

// ERROR: WRONG: Using strings for error types
if (error.type === 'validation_error') {  // Typo-prone!
  // Handle error
}
```

### 3.2 Error Propagation

**Rule:** Always wrap errors with context when re-throwing. Never lose the original error chain.

**Why:** Error chains help debug issues by showing the full call stack. Without wrapping, you lose valuable debugging information.

**DO:**

```typescript
// DONE: CORRECT: Proper error wrapping
async function processUserRegistration(data: RegisterUserDTO): Promise<User> {
  try {
    // Validate input
    const validated = validateRegistrationData(data);
    
    // Check if email exists
    const existing = await userRepository.findByEmail(validated.email);
    if (existing) {
      throw new ValidationError('Email already registered');
    }
    
    // Create user
    const user = await userRepository.create(validated);
    
    // Send welcome email
    await emailService.sendWelcome(user);
    
    return user;
  } catch (error) {
    // Wrap with context - preserves original error
    if (error instanceof ValidationError) {
      throw error; // Re-throw known errors as-is
    }
    // Wrap unknown errors with context
    throw new AppError(
      'Failed to process registration',
      'REGISTRATION_FAILED',
      500,
      error instanceof Error ? error : undefined
    );
  }
}
```

**DON'T:**

```typescript
// ERROR: WRONG: Losing error context
async function processUserRegistration(data: RegisterUserDTO): Promise<User> {
  try {
    // ... implementation
  } catch (error) {
    // Error context lost!
    throw new Error('Registration failed');
  }
}

// ERROR: WRONG: Swallowing errors
async function processData(data: unknown): Promise<Result> {
  try {
    return await riskyOperation(data);
  } catch (error) {
    // Silently ignoring error - dangerous!
    return { success: false };
  }
}
```

### 3.3 Logging Errors

**Rule:** Log errors with appropriate context. Use structured logging.

**Why:** Good logs are essential for debugging production issues. Structured logs make it easy to search and filter.

**DO:**

```typescript
// DONE: CORRECT: Structured error logging
import { logger } from './logger';

async function handleRequest(req: Request): Promise<Response> {
  try {
    return await processRequest(req);
  } catch (error) {
    logger.error('Request processing failed', {
      error: error instanceof Error ? error.message : 'Unknown error',
      stack: error instanceof Error ? error.stack : undefined,
      requestId: req.id,
      userId: req.userId,
      path: req.path,
      method: req.method,
      timestamp: new Date().toISOString(),
    });
    
    return new Response('Internal Server Error', { status: 500 });
  }
}
```

### 3.4 Never Empty Catch Blocks

**Rule:** NEVER have empty catch blocks. Always handle or re-throw the error.

**Why:** Empty catch blocks hide errors and make debugging impossible. They also indicate incomplete error handling.

**DO:**

```typescript
// DONE: CORRECT: Handle the error
try {
  await riskyOperation();
} catch (error) {
  logger.error('Operation failed', { error });
  throw error;
}

// DONE: CORRECT: Re-throw with context
try {
  await riskyOperation();
} catch (error) {
  throw new AppError('Operation failed', 'OPERATION_FAILED', 500, error instanceof Error ? error : undefined);
}

// DONE: CORRECT: Use a default value when appropriate
let result: Result;
try {
  result = await riskyOperation();
} catch {
  result = { status: 'fallback', value: defaultValue };
}
```

**DON'T:**

```typescript
// ERROR: WRONG: Empty catch block
try {
  await riskyOperation();
} catch (error) {
  // Do nothing - error silently ignored!
}

// ERROR: WRONG: Only logging but not handling
try {
  await riskyOperation();
} catch (error) {
  console.error(error);  // Logs but continues as if nothing happened
  // Should either throw or handle the error properly
}
```

---

## 4. Code Style

### 4.1 Naming Conventions

**Rule:** Use consistent naming conventions across all code.

**Why:** Consistent naming makes code readable and maintainable. New developers can quickly understand code patterns.

**TypeScript Naming:**

```typescript
// DONE: CORRECT: camelCase for variables, functions
const userName = 'John';
const isActive = true;

function calculateTotal(): number {
  return 0;
}

// DONE: CORRECT: PascalCase for classes, interfaces, types
class UserService {}
interface UserConfig {}
type ResponseStatus = 'pending' | 'completed';

// DONE: CORRECT: UPPER_SNAKE_CASE for constants
const MAX_RETRY_ATTEMPTS = 3;
const API_BASE_URL = 'https://api.example.com';

// DONE: CORRECT: Prefix booleans with is, has, can, should
const isActive = true;
const hasPermission = false;
const canEdit = true;
const shouldUpdate = false;

// ERROR: WRONG: Inconsistent naming
const user_name = 'John';  // Use camelCase
const UserService = class {};  // Classes should be PascalCase
const active = true;  // Should be isActive
```

**Go Naming:**

```go
// DONE: CORRECT: camelCase for variables, functions
userName := "John"
isActive := true

func calculateTotal() int {
    return 0
}

// DONE: CORRECT: PascalCase for exported functions, types
type UserService struct {}
func ProcessData() error {}

// DONE: CORRECT: UPPER_SNAKE_CASE for constants
const MaxRetryAttempts = 3
const ApiBaseURL = "https://api.example.com"

// DONE: CORRECT: Short names for short scope
for i := 0; i < n; i++ {  // i is fine for loop

// ERROR: WRONG: Mixed naming styles
const user_name = "John"  // Use camelCase
var UserService = class {}  // Exported but should be UserService
```

### 4.2 File Organization

**Rule:** Organize files by feature, not by type. Keep related code together.

**Why:** Feature-based organization makes it easier to navigate and understand code. It also makes it easier to extract features as modules.

**Recommended Structure:**

```
src/
├── features/
│   ├── auth/
│   │   ├── auth.controller.ts
│   │   ├── auth.service.ts
│   │   ├── auth.middleware.ts
│   │   ├── auth.types.ts
│   │   └── index.ts          # Public API
│   ├── users/
│   │   ├── users.controller.ts
│   │   ├── users.service.ts
│   │   ├── users.types.ts
│   │   └── index.ts
│   └── biometrics/
│       ├── biometrics.controller.ts
│       ├── biometrics.service.ts
│       ├── biometrics.types.ts
│       └── index.ts
├── shared/                   # Shared utilities
│   ├── errors/
│   ├── logger/
│   └── utils/
└── app.ts
```

### 4.3 Import Order

**Rule:** Organize imports in a specific order. Use a linter to enforce.

**Why:** Consistent import order makes it easier to find dependencies. It also helps identify unused imports.

**TypeScript Import Order:**

```typescript
// 1. External libraries (React, Express, etc.)
import React from 'react';
import { useState, useEffect } from 'react';
import express from 'express';
import cors from 'cors';

// 2. Internal modules (relative imports)
import { UserService } from '../services/user.service';
import { AuthMiddleware } from './auth.middleware';

// 3. Types/interfaces
import type { User, AuthConfig } from '../types';

// 4. Utilities
import { formatDate, validateEmail } from '../utils/format';

// 5. Styles (last)
import './styles.css';
```

**Go Import Order:**

```go
package main

import (
    // 1. Standard library
    "fmt"
    "os"
    "time"
    
    // 2. External packages
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    
    // 3. Internal packages (separated by blank line)
    "myproject/internal/handler"
    "myproject/internal/service"
)
```

### 4.4 Line Length Limits

**Rule:** Keep lines under 100 characters. Maximum 120 characters.

**Why:** Long lines are hard to read, especially on smaller screens. They also make diffs harder to review.

**DO:**

```typescript
// DONE: CORRECT: Break long lines
function processUserData(
  userData: UserData,
  config: ProcessingConfig,
): ProcessedUserData {
  // Implementation
}

// DONE: CORRECT: Use intermediate variables
const userFullName = `${user.firstName} ${user.lastName}`;
const isEligible = user.age >= 18 && user.isVerified;
```

**DON'T:**

```typescript
// ERROR: WRONG: Lines too long
function processUserData(userData: UserData, config: ProcessingConfig): ProcessedUserData { return { ... }; }
```

---

## 5. Testing Standards

### 5.1 Test-Driven Development (TDD)

**Rule:** Write tests before implementation. Follow the Red-Green-Refactor cycle.

**Why:** TDD ensures code is testable from the start. It also leads to better designed, more modular code.

**TDD Cycle:**

```
1. RED: Write a failing test
2. GREEN: Write minimal code to make test pass
3. REFACTOR: Improve code while keeping tests passing
4. REPEAT
```

**Example TDD Workflow:**

```typescript
// Step 1: RED - Write failing test
describe('calculateBMI', () => {
  it('should calculate BMI correctly', () => {
    const result = calculateBMI(70, 1.75);
    expect(result).toBe(22.86);
  });
});

// Step 2: GREEN - Write minimal implementation
function calculateBMI(weight: number, height: number): number {
  return 22.86; // Hardcoded for now
}

// Step 3: GREEN - Make it real
function calculateBMI(weight: number, height: number): number {
  if (weight <= 0 || height <= 0) {
    throw new Error('Invalid input');
  }
  return weight / (height * height);
}

// Step 4: REFACTOR - Add more tests and edge cases
```

### 5.2 Coverage Requirements

**Rule:** Minimum 80% code coverage for core business logic. Higher for critical paths.

**Why:** Coverage metrics help identify untested code. 80% is a good balance between thoroughness and practicality.

**Coverage Targets:**

| Category | Minimum Coverage |
|----------|----------------|
| Core business logic | 80% |
| API endpoints | 90% |
| Security/auth | 95% |
| Data validation | 90% |
| Utility functions | 80% |

**Measuring Coverage:**

```bash
# Run tests with coverage
npm run test -- --coverage

# View detailed report
npx jest --coverage --coverageReporters=text-summary
```

### 5.3 Test Naming Conventions

**Rule:** Use descriptive test names that explain what is being tested.

**Why:** Good test names serve as documentation. They make it easy to understand what behavior is being tested without reading the implementation.

**Naming Pattern:**

```typescript
describe('FunctionName', () => {
  describe('when [condition]', () => {
    it('should [expected behavior]', () => {
      // Test
    });
  });
});

// Example
describe('calculateBMI', () => {
  describe('when weight and height are valid', () => {
    it('should return correct BMI value', () => {
      expect(calculateBMI(70, 1.75)).toBeCloseTo(22.86);
    });
  });
  
  describe('when weight is zero', () => {
    it('should throw validation error', () => {
      expect(() => calculateBMI(0, 1.75)).toThrow('Weight must be positive');
    });
  });
  
  describe('when height is negative', () => {
    it('should throw validation error', () => {
      expect(() => calculateBMI(70, -1.75)).toThrow('Height must be positive');
    });
  });
});
```

### 5.4 Mock vs Real Data

**Rule:** Use mocks for external dependencies (databases, APIs). Use real data for business logic.

**Why:** Mocks make tests fast and reliable. However, business logic should be tested with real data to ensure correctness.

**DO:**

```typescript
// DONE: CORRECT: Mock external dependencies
describe('UserService', () => {
  let userService: UserService;
  let mockDatabase: jest.Mocked<Database>;
  
  beforeEach(() => {
    mockDatabase = {
      findUser: jest.fn(),
      createUser: jest.fn(),
    } as any;
    
    userService = new UserService(mockDatabase);
  });
  
  it('should create user', async () => {
    mockDatabase.createUser.mockResolvedValue({ id: '123', name: 'John' });
    
    const user = await userService.createUser({ name: 'John' });
    
    expect(user.id).toBe('123');
    expect(mockDatabase.createUser).toHaveBeenCalledWith({ name: 'John' });
  });
});

// DONE: CORRECT: Test business logic with real data
describe('calculateBMI', () => {
  it('should correctly calculate BMI with real formulas', () => {
    // Test with real data - no mocks needed
    expect(calculateBMI(70, 1.75)).toBeCloseTo(22.86, 2);
    expect(calculateBMI(80, 1.80)).toBeCloseTo(24.69, 2);
    expect(calculateBMI(50, 1.60)).toBeCloseTo(19.53, 2);
  });
});
```

---

## 6. Performance Standards

### 6.1 Connection Pooling

**Rule:** Use connection pools for all database and HTTP connections.

**Why:** Connection pooling reuses connections, reducing overhead. It also prevents overwhelming external services.

**DO:**

```typescript
// DONE: CORRECT: Database connection pooling
import { Pool } from 'pg';

const pool = new Pool({
  max: 20,                 // Maximum connections
  idleTimeoutMillis: 30000,
  connectionTimeoutMillis: 2000,
});

// Use pool.query() instead of creating new connections
async function queryUsers() {
  const result = await pool.query('SELECT * FROM users');
  return result.rows;
}

// DONE: CORRECT: HTTP agent pooling (Node.js)
import https from 'https';

const agent = new https.Agent({
  maxSockets: 25,         // Max concurrent sockets
  keepAlive: true,
  keepAliveMsecs: 30000,
});

async function fetchData(url: string) {
  const response = await fetch(url, { agent });
  return response.json();
}
```

### 6.2 Caching Strategies

**Rule:** Implement caching for expensive operations. Use appropriate cache invalidation.

**Why:** Caching dramatically improves performance. However, cache invalidation must be correct to avoid serving stale data.

**Cache Implementation:**

```typescript
// DONE: CORRECT: In-memory cache with TTL
class Cache<T> {
  private store = new Map<string, { value: T; expires: number }>();
  
  set(key: string, value: T, ttlMs: number = 60000): void {
    this.store.set(key, {
      value,
      expires: Date.now() + ttlMs,
    });
  }
  
  get(key: string): T | undefined {
    const entry = this.store.get(key);
    if (!entry) return undefined;
    
    if (Date.now() > entry.expires) {
      this.store.delete(key);
      return undefined;
    }
    
    return entry.value;
  }
}

// DONE: CORRECT: Redis cache for distributed systems
import Redis from 'ioredis';

const redis = new Redis();

async function getCachedUser(id: string): Promise<User | null> {
  const cacheKey = `user:${id}`;
  
  // Try cache first
  const cached = await redis.get(cacheKey);
  if (cached) {
    return JSON.parse(cached);
  }
  
  // Fetch from database
  const user = await db.findUser(id);
  if (user) {
    // Cache for 5 minutes
    await redis.setex(cacheKey, 300, JSON.stringify(user));
  }
  
  return user;
}
```

### 6.3 Lazy Loading

**Rule:** Use lazy loading for heavy imports and routes.

**Why:** Lazy loading reduces initial bundle size and improves startup time. Users only download code they actually use.

**DO:**

```typescript
// DONE: CORRECT: Lazy load heavy modules
const HeavyComponent = React.lazy(() => import('./HeavyComponent'));

// DONE: CORRECT: Lazy load routes
const AdminDashboard = lazy(() => import('./pages/AdminDashboard'));
const ReportsPage = lazy(() => import('./pages/ReportsPage'));

function App() {
  return (
    <Routes>
      <Route path="/" element={<Home />} />
      <Route 
        path="/admin" 
        element={
          <Suspense fallback={<Loading />}>
            <AdminDashboard />
          </Suspense>
        } 
      />
    </Routes>
  );
}

// DONE: CORRECT: Lazy load data
async function getData() {
  const { heavyFunction } = await import('./heavyModule');
  return heavyFunction();
}
```

### 6.4 Bundle Size Limits

**Rule:** Keep initial bundle under 200KB (compressed). Lazy load everything else.

**Why:** Bundle size directly affects load time. Large bundles frustrate users and hurt SEO.

**Monitoring:**

```json
// package.json
{
  "scripts": {
    "analyze": "webpack-bundle-analyzer --port 8888"
  }
}
```

**Budget Configuration (next.config.js):**

```javascript
module.exports = {
  webpack: (config, { isServer }) => {
    config.optimization = {
      ...config.optimization,
      splitChunks: {
        chunks: 'all',
        cacheGroups: {
          vendor: {
            test: /[\\/]node_modules[\\/]/,
            name: 'vendors',
            chunks: 'all',
          },
        },
      },
    };
    return config;
  },
};
```

---

## 7. Security Standards

### 7.1 Input Validation

**Rule:** Validate ALL input at the application boundary. Never trust user input.

**Why:** Invalid input is the leading cause of security vulnerabilities. Validation prevents injection attacks, data corruption, and crashes.

**DO:**

```typescript
// DONE: CORRECT: Validate at boundaries
import { z } from 'zod';

const CreateUserSchema = z.object({
  email: z.string().email('Invalid email format'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
  name: z.string().min(1, 'Name is required').max(100),
  age: z.number().int().min(0).max(150).optional(),
});

type CreateUserDTO = z.infer<typeof CreateUserSchema>;

async function createUserHandler(req: Request): Promise<Response> {
  // Validate input immediately at the boundary
  const result = CreateUserSchema.safeParse(req.body);
  
  if (!result.success) {
    return new Response(
      JSON.stringify({ errors: result.error.flatten() }),
      { status: 400 }
    );
  }
  
  // Process validated data
  const user = await userService.create(result.data);
  return Response.json(user);
}

// DONE: CORRECT: Validate on API layer AND service layer
function calculateBMI(weight: number, height: number): number {
  // Defense in depth - validate at every layer
  if (typeof weight !== 'number' || typeof height !== 'number') {
    throw new ValidationError('Weight and height must be numbers');
  }
  if (!Number.isFinite(weight) || !Number.isFinite(height)) {
    throw new ValidationError('Weight and height must be finite numbers');
  }
  if (weight <= 0 || height <= 0) {
    throw new ValidationError('Weight and height must be positive');
  }
  
  return weight / (height * height);
}
```

### 7.2 SQL Injection Prevention

**Rule:** NEVER concatenate user input into SQL queries. Use parameterized queries or ORMs.

**Why:** SQL injection is one of the most dangerous web vulnerabilities. It allows attackers to access, modify, or delete data.

**DO:**

```typescript
// DONE: CORRECT: Use parameterized queries
const result = await pool.query(
  'SELECT * FROM users WHERE email = $1',
  [email]  // Parameterized - safe!
);

// DONE: CORRECT: Use ORM (Prisma, Drizzle, etc.)
const user = await prisma.user.findUnique({
  where: { email },
});

// DONE: CORRECT: Use query builder with proper escaping
const users = await db
  .select()
  .from(usersTable)
  .where(eq(usersTable.email, email));
```

**DON'T:**

```typescript
// ERROR: WRONG: SQL injection vulnerable!
const query = `SELECT * FROM users WHERE email = '${email}'`;
// If email = "' OR '1'='1", attacker gets all users!
```

### 7.3 XSS Prevention

**Rule:** Escape all user input before rendering. Use Content Security Policy.

**Why:** Cross-site scripting (XSS) allows attackers to inject malicious scripts into web pages.

**DO:**

```typescript
// DONE: CORRECT: Use framework's auto-escaping
// React, Vue, Angular automatically escape by default
function UserName({ name }: { name: string }) {
  return <span>{name}</span>;  // Automatically escaped!
}

// DONE: CORRECT: Use safe DOM APIs
element.textContent = userInput;  // Automatically escaped
element.innerText = userInput;    // Automatically escaped

// ERROR: WRONG: Never use innerHTML with user input
element.innerHTML = userInput;  // XSS vulnerability!
```

### 7.4 Secret Management

**Rule:** Never commit secrets to version control. Use environment variables or secret management services.

**Why:** Exposed secrets lead to unauthorized access, data breaches, and financial loss.

**DO:**

```typescript
// DONE: CORRECT: Use environment variables
import dotenv from 'dotenv';
dotenv.config();

const apiKey = process.env.API_KEY;
if (!apiKey) {
  throw new Error('API_KEY environment variable is required');
}

// DONE: CORRECT: Use secret management services
import { SecretManager } from '@aws-sdk/client-secrets-manager';

async function getSecret(secretName: string): Promise<string> {
  const client = new SecretManager({ region: 'us-east-1' });
  const result = await client.getSecretValue({ SecretId: secretName });
  return result.SecretString!;
}
```

**DON'T:**

```typescript
// ERROR: WRONG: Hardcoded secrets
const API_KEY = 'sk-1234567890abcdef';  // NEVER commit this!

// ERROR: WRONG: Secrets in config files
// config.json
{
  "apiKey": "sk-1234567890abcdef"  // NEVER commit this!
}
```

---

## 8. Documentation Standards

### 8.1 Code Comments

**Rule:** Comment WHY, not WHAT. Code should be self-documenting.

**Why:** Good comments explain reasoning, not implementation. They help future developers understand decisions.

**DO:**

```typescript
// DONE: CORRECT: Explain WHY
// Using exponential backoff because the external API 
// has rate limiting and returns 429 on consecutive failures
async function retryWithBackoff<T>(
  fn: () => Promise<T>,
  maxRetries = 3
): Promise<T> {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await fn();
    } catch (error) {
      if (i === maxRetries - 1) throw error;
      await sleep(Math.pow(2, i) * 1000);
    }
  }
  throw new Error('Unreachable');
}
```

**DON'T:**

```typescript
// ERROR: WRONG: Explain WHAT (obvious)
// Increment i by 1
i++;

// ERROR: WRONG: Commented-out code
// const oldCode = value;  // DELETE THIS LATER?
```

---

## 9. Git Workflow

### 9.1 Commit Messages

**Rule:** Use conventional commits format.

**Why:** Conventional commits enable automated changelog generation and help understand commit history.

**Format:**

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Formatting
- `refactor`: Code restructure
- `test`: Tests
- `chore`: Maintenance

**Examples:**

```bash
# Good commit messages
git commit -m "feat(auth): add password reset functionality"
git commit -m "fix(user): handle null email in user service"
git commit -m "docs(api): update API documentation for v2"
git commit -m "refactor(biometrics): improve face recognition algorithm"
git commit -m "test(auth): add unit tests for login flow"
```

---

## 10. CI/CD Standards

### 10.1 Required Checks

**Rule:** All PRs must pass before merging.

**Required Checks:**
- DONE: Linting passes
- DONE: TypeScript compiles without errors
- DONE: All tests pass
- DONE: Coverage meets minimum threshold
- DONE: Security scan passes

### 10.2 Pipeline Stages

```
1. Lint     → ESLint, go fmt
2. TypeCheck → tsc --noEmit, go build
3. Test     → Jest, Go test
4. Coverage → Coverage report
5. Security → Security scan
6. Build    → Docker build
7. Deploy   → Deploy to staging/production
```

---

## References

- [TypeScript Handbook](https://www.typescriptlang.org/docs/)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Airbnb JavaScript Style Guide](https://github.com/airbnb/javascript)
- [Google TypeScript Style Guide](https://google.github.io/styleguide/tsguide.html)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Best Practices 2026](https://example.com/best-practices-2026)

---

**Document Control:**
- Created: 2026-02-20
- Version: 1.0
- Owner: BIOMETRICS Development Team
- Review Cycle: Quarterly
