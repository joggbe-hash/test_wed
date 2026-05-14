import React from 'react';
import Layout from '../components/Layout';

function FreqPage() {
  const listItems = Array.from({ length: 6 });

  return (
    <Layout sidebarType="freq" fabIcon="+">
      {listItems.map((_, index) => (
        <div key={index} className="freq-list-item"></div>
      ))}
    </Layout>
  );
}

export default FreqPage;
