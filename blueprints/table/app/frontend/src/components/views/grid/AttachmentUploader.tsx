import { useState, useRef, useCallback } from 'react';

interface Attachment {
  id?: string;
  filename: string;
  url: string;
  mime_type: string;
  size: number;
  width?: number;
  height?: number;
}

interface AttachmentUploaderProps {
  existingAttachments: Attachment[];
  onUpload: (attachments: Attachment[]) => void;
  onDelete: (attachment: Attachment) => void;
  onClose: () => void;
  maxFiles?: number;
  acceptedTypes?: string[];
}

interface UploadingFile {
  id: string;
  file: File;
  progress: number;
  error?: string;
  preview?: string;
}

export function AttachmentUploader({
  existingAttachments,
  onUpload,
  onDelete,
  onClose,
  maxFiles = 10,
  acceptedTypes = ['image/*', 'application/pdf', '.doc', '.docx', '.xls', '.xlsx'],
}: AttachmentUploaderProps) {
  const [uploadingFiles, setUploadingFiles] = useState<UploadingFile[]>([]);
  const [isDragging, setIsDragging] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Generate unique ID
  const generateId = () => `upload-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

  // Format file size
  const formatSize = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  // Handle file selection
  const handleFiles = useCallback(async (files: FileList | File[]) => {
    const fileArray = Array.from(files);
    const remaining = maxFiles - existingAttachments.length - uploadingFiles.length;

    if (fileArray.length > remaining) {
      alert(`You can only upload ${remaining} more file(s)`);
      return;
    }

    // Create uploading entries with previews
    const newUploads: UploadingFile[] = await Promise.all(
      fileArray.map(async (file) => {
        const upload: UploadingFile = {
          id: generateId(),
          file,
          progress: 0,
        };

        // Generate preview for images
        if (file.type.startsWith('image/')) {
          upload.preview = await new Promise((resolve) => {
            const reader = new FileReader();
            reader.onloadend = () => resolve(reader.result as string);
            reader.readAsDataURL(file);
          });
        }

        return upload;
      })
    );

    setUploadingFiles((prev) => [...prev, ...newUploads]);

    // Simulate upload for each file
    for (const upload of newUploads) {
      await uploadFile(upload);
    }
  }, [existingAttachments.length, uploadingFiles.length, maxFiles]);

  // Simulate file upload (replace with actual API call)
  const uploadFile = async (upload: UploadingFile) => {
    const { file } = upload;

    try {
      // Simulate progress
      for (let progress = 0; progress <= 100; progress += 10) {
        await new Promise((r) => setTimeout(r, 100));
        setUploadingFiles((prev) =>
          prev.map((u) => (u.id === upload.id ? { ...u, progress } : u))
        );
      }

      // Create attachment object
      const attachment: Attachment = {
        id: generateId(),
        filename: file.name,
        url: upload.preview || URL.createObjectURL(file),
        mime_type: file.type,
        size: file.size,
      };

      // Get image dimensions
      if (file.type.startsWith('image/') && upload.preview) {
        const img = new Image();
        await new Promise((resolve) => {
          img.onload = resolve;
          img.src = upload.preview!;
        });
        attachment.width = img.naturalWidth;
        attachment.height = img.naturalHeight;
      }

      // Add to existing attachments
      onUpload([...existingAttachments, attachment]);

      // Remove from uploading list
      setUploadingFiles((prev) => prev.filter((u) => u.id !== upload.id));
    } catch (error) {
      setUploadingFiles((prev) =>
        prev.map((u) =>
          u.id === upload.id
            ? { ...u, error: 'Upload failed', progress: 0 }
            : u
        )
      );
    }
  };

  // Handle drag events
  const handleDragEnter = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(true);
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);

    const files = e.dataTransfer.files;
    if (files.length > 0) {
      handleFiles(files);
    }
  };

  // Handle file input change
  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      handleFiles(e.target.files);
    }
  };

  // Retry failed upload
  const retryUpload = (upload: UploadingFile) => {
    setUploadingFiles((prev) =>
      prev.map((u) => (u.id === upload.id ? { ...u, error: undefined, progress: 0 } : u))
    );
    uploadFile(upload);
  };

  // Cancel upload
  const cancelUpload = (uploadId: string) => {
    setUploadingFiles((prev) => prev.filter((u) => u.id !== uploadId));
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-lg max-h-[80vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-slate-200">
          <h3 className="text-lg font-semibold text-slate-900">Attachments</h3>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-slate-600"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto p-4">
          {/* Drop zone */}
          <div
            onDragEnter={handleDragEnter}
            onDragLeave={handleDragLeave}
            onDragOver={handleDragOver}
            onDrop={handleDrop}
            onClick={() => fileInputRef.current?.click()}
            className={`border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors ${
              isDragging
                ? 'border-primary bg-primary-50'
                : 'border-slate-300 hover:border-slate-400'
            }`}
          >
            <input
              ref={fileInputRef}
              type="file"
              multiple
              accept={acceptedTypes.join(',')}
              onChange={handleInputChange}
              className="hidden"
            />
            <svg
              className={`w-12 h-12 mx-auto mb-4 ${
                isDragging ? 'text-primary' : 'text-slate-400'
              }`}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
            </svg>
            <p className="text-sm text-slate-600 mb-1">
              {isDragging ? 'Drop files here' : 'Drag and drop files here'}
            </p>
            <p className="text-xs text-slate-400">
              or click to browse
            </p>
          </div>

          {/* Uploading files */}
          {uploadingFiles.length > 0 && (
            <div className="mt-4 space-y-2">
              <p className="text-sm font-medium text-slate-700">Uploading</p>
              {uploadingFiles.map((upload) => (
                <div
                  key={upload.id}
                  className="flex items-center gap-3 p-3 bg-slate-50 rounded-lg"
                >
                  {/* Preview */}
                  <div className="w-10 h-10 rounded overflow-hidden flex-shrink-0 bg-slate-200">
                    {upload.preview ? (
                      <img
                        src={upload.preview}
                        alt=""
                        className="w-full h-full object-cover"
                      />
                    ) : (
                      <div className="w-full h-full flex items-center justify-center">
                        <svg className="w-5 h-5 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
                        </svg>
                      </div>
                    )}
                  </div>

                  {/* Info and progress */}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm text-slate-700 truncate">
                      {upload.file.name}
                    </p>
                    {upload.error ? (
                      <p className="text-xs text-red-600">{upload.error}</p>
                    ) : (
                      <div className="mt-1 h-1.5 bg-slate-200 rounded-full overflow-hidden">
                        <div
                          className="h-full bg-primary transition-all duration-300"
                          style={{ width: `${upload.progress}%` }}
                        />
                      </div>
                    )}
                  </div>

                  {/* Actions */}
                  {upload.error ? (
                    <button
                      onClick={() => retryUpload(upload)}
                      className="text-primary hover:text-primary-dark text-sm"
                    >
                      Retry
                    </button>
                  ) : (
                    <button
                      onClick={() => cancelUpload(upload.id)}
                      className="text-slate-400 hover:text-slate-600"
                    >
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                      </svg>
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}

          {/* Existing attachments */}
          {existingAttachments.length > 0 && (
            <div className="mt-4 space-y-2">
              <p className="text-sm font-medium text-slate-700">
                Attached files ({existingAttachments.length})
              </p>
              {existingAttachments.map((attachment, index) => (
                <div
                  key={attachment.id || index}
                  className="flex items-center gap-3 p-3 bg-slate-50 rounded-lg"
                >
                  {/* Preview */}
                  <div className="w-10 h-10 rounded overflow-hidden flex-shrink-0 bg-slate-200">
                    {attachment.mime_type?.startsWith('image/') ? (
                      <img
                        src={attachment.url}
                        alt=""
                        className="w-full h-full object-cover"
                      />
                    ) : (
                      <div className="w-full h-full flex items-center justify-center">
                        <svg className="w-5 h-5 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
                        </svg>
                      </div>
                    )}
                  </div>

                  {/* Info */}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm text-slate-700 truncate">
                      {attachment.filename}
                    </p>
                    <p className="text-xs text-slate-400">
                      {formatSize(attachment.size)}
                    </p>
                  </div>

                  {/* Delete button */}
                  <button
                    onClick={() => onDelete(attachment)}
                    className="text-slate-400 hover:text-red-600"
                    title="Delete"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2 p-4 border-t border-slate-200">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm text-slate-700 hover:bg-slate-100 rounded-lg"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
