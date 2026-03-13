import { EngineSchema, getKustoWorker } from '@kusto/monaco-kusto';
import { useCallback, useEffect, useState } from 'react';

import { CodeEditor, Monaco, MonacoEditor } from '@grafana/ui';

import { AzureQueryEditorFieldProps } from '../../types/types';

import { setKustoQuery } from './setQueryValue';

import { createStructuredLogger } from '@grafana/data';
const structuredLogger = createStructuredLogger('public/app/plugins/datasource/azuremonitor/components/LogsQueryEditor/QueryField');

interface MonacoEditorValues {
  editor: MonacoEditor;
  monaco: Monaco;
}

const QueryField = ({ query, onQueryChange, schema }: AzureQueryEditorFieldProps) => {
  const [monaco, setMonaco] = useState<MonacoEditorValues | undefined>();

  useEffect(() => {
    if (!schema || !monaco) {
      return;
    }

    const setupEditor = async ({ monaco, editor }: MonacoEditorValues, schema: EngineSchema) => {
      try {
        const model = editor.getModel();
        if (model) {
          const kustoWorker = await getKustoWorker();
          const kustoMode = await kustoWorker(model?.uri);
          await kustoMode.setSchema(schema);
        }
      } catch (err) {
        structuredLogger.error(err);
      }
    };

    setupEditor(monaco, schema).catch((err) => structuredLogger.error(err));
  }, [schema, monaco]);

  const handleEditorMount = useCallback((editor: MonacoEditor, monaco: Monaco) => {
    setMonaco({ monaco, editor });
  }, []);

  const onChange = useCallback(
    (newQuery: string) => {
      onQueryChange(setKustoQuery(query, newQuery));
    },
    [onQueryChange, query]
  );

  return (
    <CodeEditor
      value={query.azureLogAnalytics?.query ?? ''}
      language="kusto"
      height={200}
      width="100%"
      showMiniMap={false}
      onBlur={onChange}
      onSave={onChange}
      onEditorDidMount={handleEditorMount}
    />
  );
};

export default QueryField;
