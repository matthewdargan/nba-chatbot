from langchain_community.document_loaders.csv_loader import CSVLoader
from langchain_community.embeddings import OllamaEmbeddings

loader = CSVLoader(file_path='stats.csv')
data = loader.load()
print(data)
embeddings = OllamaEmbeddings(model="llama2:7b")
doc_result = embeddings.embed_documents([data])
print(doc_result)