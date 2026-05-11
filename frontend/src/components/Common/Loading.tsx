import React from 'react';

interface LoadingProps {
  message?: string;
  fullscreen?: boolean;
}

const Loading: React.FC<LoadingProps> = ({ message = 'Loading...', fullscreen = false }) => {
  const className = fullscreen ? 'loading' : 'flex justify-center items-center py-12';

  return (
    <div className={className}>
      <div className="text-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
        <p className="text-gray-600">{message}</p>
      </div>
    </div>
  );
};

export default Loading;
