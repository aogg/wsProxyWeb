module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'jsdom',
  roots: ['<rootDir>/src/tests'],
  testMatch: ['**/*.test.ts'],
  moduleFileExtensions: ['ts', 'js', 'json'],
  collectCoverageFrom: [
    'src/libs/**/*.ts',
    '!src/libs/**/*.d.ts',
  ],
  coverageDirectory: 'coverage',
  coverageReporters: ['text', 'lcov'],
  verbose: true,
  // 模拟浏览器API
  setupFilesAfterEnv: ['<rootDir>/src/tests/setup.ts'],
  // 忽略转换的模块
  transformIgnorePatterns: [
    'node_modules/(?!(node-fetch)/)',
  ],
};
