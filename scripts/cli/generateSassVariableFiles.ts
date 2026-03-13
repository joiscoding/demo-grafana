import { writeFile } from 'node:fs/promises';
import { resolve } from 'path';

import { createTheme } from '@grafana/data';

import { darkThemeVarsTemplate } from './themeTemplates/_variables.dark.scss.tmpl';
import { lightThemeVarsTemplate } from './themeTemplates/_variables.light.scss.tmpl';
import { commonThemeVarsTemplate } from './themeTemplates/_variables.scss.tmpl';

import * as structuredLogging from '../helpers/structuredLogging';
const { createStructuredLogger } = structuredLogging;
const structuredLogger = createStructuredLogger('scripts/cli/generateSassVariableFiles');

const darkThemeVariablesPath = resolve(__dirname, 'public', 'sass', '_variables.dark.generated.scss');
const lightThemeVariablesPath = resolve(__dirname, 'public', 'sass', '_variables.light.generated.scss');
const defaultThemeVariablesPath = resolve(__dirname, 'public', 'sass', '_variables.generated.scss');

async function writeVariablesFile(path: string, data: string) {
  try {
    await writeFile(path, data);
  } catch (error) {
    structuredLogger.error('\nWriting SASS variable files failed', error);
    process.exit(1);
  }
}

async function generateSassVariableFiles() {
  const darkTheme = createTheme();
  const lightTheme = createTheme({ colors: { mode: 'light' } });
  try {
    await writeVariablesFile(darkThemeVariablesPath, darkThemeVarsTemplate(darkTheme));
    await writeVariablesFile(lightThemeVariablesPath, lightThemeVarsTemplate(lightTheme));
    await writeVariablesFile(defaultThemeVariablesPath, commonThemeVarsTemplate(darkTheme));
  } catch (error) {
    structuredLogger.error('\nWriting SASS variable files failed', error);
    process.exit(1);
  }
}

generateSassVariableFiles();
